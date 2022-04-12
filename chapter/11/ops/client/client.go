package client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto"
	mpb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto/jaeger/model"
)

// Ops is a client for interacting with the Ops service.
type Ops struct {
	client pb.OpsClient
	conn   *grpc.ClientConn
}

// New is the constructor for Client. addr is the server's [host]:[port].
func New(addr string) (*Ops, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Ops{
		client: pb.NewOpsClient(conn),
		conn:   conn,
	}, nil
}

type callOptions struct {
	l *listTracesOpts
	a *alertsOpts
}

type listTracesOpts struct {
	operation string
	tags      []string
	start     time.Time
	end       time.Time
	limit     int32
}

type alertsOpts struct {
	labels   []string
	activeAt time.Time
	states   []string
}

func (l *listTracesOpts) defaults() {
	l.limit = 10
	l.end = time.Now().Add(5 * time.Second)
}

// CallOption is an option for method.
type CallOption func(o *callOptions) error

// WithStart sets the minimum time a trace had to start in order to be included.
func WithStart(t time.Time) CallOption {
	return func(o *callOptions) error {
		if o.l == nil {
			return fmt.Errorf("WithStart can only be used on ListTraces()")
		}
		o.l.start = t
		return nil
	}
}

// Withend sets the maximum time a trace had to start in order to be included.
func WithEnd(t time.Time) CallOption {
	return func(o *callOptions) error {
		if o.l == nil {
			return fmt.Errorf("WithEnd can only be used on ListTraces()")
		}
		o.l.end = t
		return nil
	}
}

// WithLimt sets the maximum amount of return values. When using Cassandra, this is
// not a limit, it is some weird value where increasing it can increase how deep
// cassandra searches for traces. In that case, setting 20 might return 10 results
// but setting 100 might return 20.
func WithLimit(i int32) CallOption {
	return func(o *callOptions) error {
		if o.l == nil {
			return fmt.Errorf("WithLimit can only be used on ListTraces()")
		}
		o.l.limit = i
		return nil
	}
}

// WithOperation restricts traces to ones that have this operation.
func WithOperation(s string) CallOption {
	return func(o *callOptions) error {
		if o.l == nil {
			return fmt.Errorf("WithOperation can only be used on ListTraces()")
		}
		o.l.operation = s
		return nil
	}
}

// WithTags restricts resuts to ones that have all these tags.
func WithTags(tags []string) CallOption {
	return func(o *callOptions) error {
		if o.l == nil {
			return fmt.Errorf("WithTags can only be used on ListTraces()")
		}
		o.l.tags = tags
		return nil
	}
}

// TraceItem details information on a trace.
type TraceItem struct {
	// ID is the ID of the trace in hex form.
	ID string
	// Start is the start time of the trace.
	Start time.Time
}

// ListTraces lists traces for the Petstore. By default it pulls the latest 10 items.
func (o *Ops) ListTraces(ctx context.Context, options ...CallOption) ([]TraceItem, error) {
	opts := callOptions{l: &listTracesOpts{}}
	opts.l.defaults()
	for _, o := range options {
		o(&opts)
	}

	req := &pb.ListTracesReq{
		Service:     "petstore",
		Operation:   opts.l.operation,
		Tags:        opts.l.tags,
		Start:       opts.l.start.UnixNano(),
		End:         opts.l.end.UnixNano(),
		SearchDepth: opts.l.limit,
	}

	resp, err := o.client.ListTraces(ctx, req)
	if err != nil {
		return nil, err
	}

	items := make([]TraceItem, 0, len(resp.Traces))
	for _, ti := range resp.Traces {
		items = append(items, TraceItem{ID: ti.Id, Start: time.Unix(0, ti.Start)})
	}

	return items, nil
}

type TraceData struct {
	ID         string
	Operations []string
	Errors     []string
	Tags       []string
	Duration   time.Time
}

// ShowTrace returns the Jaeger URL that is going to have the trace.
func (o *Ops) ShowTrace(ctx context.Context, id string) (*pb.ShowTraceResp, error) {
	resp, err := o.client.ShowTrace(ctx, &pb.ShowTraceReq{Id: id})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type Log struct {
	Time  time.Time
	Key   string
	Value string
}

func (o *Ops) ShowLogs(ctx context.Context, id string) ([]Log, error) {
	resp, err := o.client.ShowLogs(ctx, &pb.ShowLogsReq{Id: id})
	if err != nil {
		return nil, err
	}
	if resp.Id == "" {
		return nil, fmt.Errorf("no trace with ID(%s) was found", id)
	}
	logs := make([]Log, 0, len(resp.Logs))
	for _, l := range resp.Logs {
		t := l.Timestamp.AsTime().UTC()
		var v string
		for _, f := range l.Fields {
			switch f.VType {
			case mpb.ValueType_BINARY:
				v = string(f.VBinary)
			case mpb.ValueType_BOOL:
				if f.VBool {
					v = "true"
				} else {
					v = "false"
				}
			case mpb.ValueType_FLOAT64:
				v = strconv.FormatFloat(f.VFloat64, 'e', 2, 64)
			case mpb.ValueType_INT64:
				v = strconv.FormatInt(f.VInt64, 10)
			case mpb.ValueType_STRING:
				v = f.VStr
			default:
				v = fmt.Sprintf("unsupported type: %T", f.VType)
			}
			logs = append(logs, Log{Time: t, Key: f.Key, Value: v})
		}
	}
	return logs, nil
}

// ChangeSampling is used to change the sampling rate of the service.
func (o *Ops) ChangeSampling(ctx context.Context, sampler *pb.ChangeSamplingReq) error {
	_, err := o.client.ChangeSampling(ctx, sampler)
	if err != nil {
		return err
	}
	return nil
}

// DeployedVersion will return the deployed version of the application according
// to Prometheus.
func (o *Ops) DeployedVersion(ctx context.Context) (string, error) {
	resp, err := o.client.DeployedVersion(ctx, &pb.DeployedVersionReq{})
	if err != nil {
		return "", err
	}
	return resp.Version, nil
}

// WithLabels restricts alerts to ones that have all these labels.
func WithLabels(labels []string) CallOption {
	return func(o *callOptions) error {
		if o.a == nil {
			return fmt.Errorf("WithLabels can only be used on Alerts()")
		}
		o.a.labels = labels
		return nil
	}
}

// WithActiveAt restrics alerts to ones from this time to now.
func WithActiveAt(t time.Time) CallOption {
	return func(o *callOptions) error {
		if o.a == nil {
			return fmt.Errorf("WithActiveAt can only be used on Alerts()")
		}
		o.a.activeAt = t
		return nil
	}
}

// WithStates restrics alerts to ones that have one of these states.
func WithStates(states []string) CallOption {
	return func(o *callOptions) error {
		if o.a == nil {
			return fmt.Errorf("WithStates can only be used on Alerts()")
		}
		o.a.states = states
		return nil
	}
}

// Alert represents a Prometheus alert.
type Alert struct {
	// State is the state of the alert.
	State string
	// Value is the value of the alert.
	Value string
	// ActiveAt was when the alert started.
	ActiveAt time.Time
}

func (a *Alert) fromProto(p *pb.Alert) {
	a.State = p.State
	a.Value = p.Value
	a.ActiveAt = time.Unix(0, p.ActiveAt)
}

// Alerts returns a list of Prometheus alerts that are firing.
func (o *Ops) Alerts(ctx context.Context, options ...CallOption) ([]Alert, error) {
	opts := callOptions{a: &alertsOpts{}}
	for _, o := range options {
		o(&opts)
	}

	req := &pb.AlertsReq{
		Labels:   opts.a.labels,
		ActiveAt: opts.a.activeAt.UnixNano(),
		States:   opts.a.states,
	}

	resp, err := o.client.Alerts(ctx, req)
	if err != nil {
		return nil, err
	}

	alerts := make([]Alert, 0, len(resp.Alerts))
	for _, p := range resp.Alerts {
		a := Alert{}
		a.fromProto(p)
		alerts = append(alerts, a)
	}
	return alerts, nil
}
