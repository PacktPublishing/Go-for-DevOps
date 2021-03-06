// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.18.0
// source: diskerase.proto

package diskerase

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Status details the status of a Block or Job.
type Status int32

const (
	// Indicates that there is some bug
	// as the Status for the object was not set.
	Status_StatusUnknown Status = 0
	// The WorkReq, Block or Job has not started execution.
	Status_StatusNotStarted Status = 1
	// The WorkReq, Block or Job is currently executing.
	Status_StatusRunning Status = 2
	// The WorkReq, Block or Job has failed.
	Status_StatusFailed Status = 3
	// The WorkReq, Block or Job has completed.
	Status_StatusCompleted Status = 4
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0: "StatusUnknown",
		1: "StatusNotStarted",
		2: "StatusRunning",
		3: "StatusFailed",
		4: "StatusCompleted",
	}
	Status_value = map[string]int32{
		"StatusUnknown":    0,
		"StatusNotStarted": 1,
		"StatusRunning":    2,
		"StatusFailed":     3,
		"StatusCompleted":  4,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_diskerase_proto_enumTypes[0].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_diskerase_proto_enumTypes[0]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{0}
}

// WorkReq is the definition of some work to be done by the system.
type WorkReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This is used to describe the work to be done. This name
	// must be authorized by having a policy with the same name
	// in the server's policies.json fie.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// A description of what this is doing.
	Desc string `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	// These are groupings of Jobs. Each block is executed one at
	// a time.
	Blocks []*Block `protobuf:"bytes,3,rep,name=blocks,proto3" json:"blocks,omitempty"`
}

func (x *WorkReq) Reset() {
	*x = WorkReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WorkReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WorkReq) ProtoMessage() {}

func (x *WorkReq) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WorkReq.ProtoReflect.Descriptor instead.
func (*WorkReq) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{0}
}

func (x *WorkReq) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *WorkReq) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *WorkReq) GetBlocks() []*Block {
	if x != nil {
		return x.Blocks
	}
	return nil
}

// WorkResp details the ID that will be used to refer to a submitted WorkReq.
type WorkResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This is the unique ID for this WorkReq.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *WorkResp) Reset() {
	*x = WorkResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WorkResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WorkResp) ProtoMessage() {}

func (x *WorkResp) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WorkResp.ProtoReflect.Descriptor instead.
func (*WorkResp) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{1}
}

func (x *WorkResp) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// Block is a grouping of Jobs that will be executed concurrently
// at some rate.
type Block struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This describes what the Block is doing.
	Desc string `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty"`
	// The amount of concurrency executions. < 1 will default to 1.
	RateLimit int32 `protobuf:"varint,2,opt,name=rate_limit,json=rateLimit,proto3" json:"rate_limit,omitempty"`
	// The Jobs to to execute in this Block.
	Jobs []*Job `protobuf:"bytes,3,rep,name=jobs,proto3" json:"jobs,omitempty"`
}

func (x *Block) Reset() {
	*x = Block{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Block) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Block) ProtoMessage() {}

func (x *Block) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Block.ProtoReflect.Descriptor instead.
func (*Block) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{2}
}

func (x *Block) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *Block) GetRateLimit() int32 {
	if x != nil {
		return x.RateLimit
	}
	return 0
}

func (x *Block) GetJobs() []*Job {
	if x != nil {
		return x.Jobs
	}
	return nil
}

// Job refers to a Job action that is defined on the server.
type Job struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This is the name of the Job, which must be registered on
	// the server.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// This is a description of what the job is doing.
	Desc string `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	// A mapping of key/value arguments. While the value is a string,
	// it can represent non-string data and will be converted by the
	// Job on the server. See the Job definition for a list of arguments
	// that are mandatory and optional.
	Args map[string]string `protobuf:"bytes,3,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Job) Reset() {
	*x = Job{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Job) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Job) ProtoMessage() {}

func (x *Job) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Job.ProtoReflect.Descriptor instead.
func (*Job) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{3}
}

func (x *Job) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Job) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *Job) GetArgs() map[string]string {
	if x != nil {
		return x.Args
	}
	return nil
}

// ExecReq is used to tell the server to execute a WorkReq
// that was previously submitted.
type ExecReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This is the unique ID of the WorkReq given back
	// by WorkResp.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *ExecReq) Reset() {
	*x = ExecReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecReq) ProtoMessage() {}

func (x *ExecReq) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecReq.ProtoReflect.Descriptor instead.
func (*ExecReq) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{4}
}

func (x *ExecReq) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// ExecResp is the response from an ExecReq.
type ExecResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ExecResp) Reset() {
	*x = ExecResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecResp) ProtoMessage() {}

func (x *ExecResp) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecResp.ProtoReflect.Descriptor instead.
func (*ExecResp) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{5}
}

// StatusReq requests a status update from the server.
type StatusReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The unique ID of the WorkReq.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *StatusReq) Reset() {
	*x = StatusReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusReq) ProtoMessage() {}

func (x *StatusReq) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusReq.ProtoReflect.Descriptor instead.
func (*StatusReq) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{6}
}

func (x *StatusReq) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// StatusResp is the status of WorkReq.
type StatusResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the WorkReq.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The description of the WorkReq.
	Desc string `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	// The overall status of the WorkReq.
	Status Status `protobuf:"varint,3,opt,name=status,proto3,enum=diskerase.Status" json:"status,omitempty"`
	// The status information on the Blocks.
	Blocks []*BlockStatus `protobuf:"bytes,4,rep,name=blocks,proto3" json:"blocks,omitempty"`
	// If we are SatusFailed or StatusCompleted, if
	// there were any errors when run.
	HadErrors bool `protobuf:"varint,5,opt,name=had_errors,json=hadErrors,proto3" json:"had_errors,omitempty"`
	// If the WorkReq was stopped with emergency stop.
	WasEsStopped bool `protobuf:"varint,6,opt,name=was_es_stopped,json=wasEsStopped,proto3" json:"was_es_stopped,omitempty"`
}

func (x *StatusResp) Reset() {
	*x = StatusResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResp) ProtoMessage() {}

func (x *StatusResp) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusResp.ProtoReflect.Descriptor instead.
func (*StatusResp) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{7}
}

func (x *StatusResp) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *StatusResp) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *StatusResp) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_StatusUnknown
}

func (x *StatusResp) GetBlocks() []*BlockStatus {
	if x != nil {
		return x.Blocks
	}
	return nil
}

func (x *StatusResp) GetHadErrors() bool {
	if x != nil {
		return x.HadErrors
	}
	return false
}

func (x *StatusResp) GetWasEsStopped() bool {
	if x != nil {
		return x.WasEsStopped
	}
	return false
}

// BlockStatus holds the status of block execution.
type BlockStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The description of the block.
	Desc string `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty"`
	// The status of the block.
	Status Status `protobuf:"varint,2,opt,name=status,proto3,enum=diskerase.Status" json:"status,omitempty"`
	// If there any errors in Jobs in the Block.
	HasError bool `protobuf:"varint,3,opt,name=has_error,json=hasError,proto3" json:"has_error,omitempty"`
	// The status of Jobs in the Block.
	Jobs []*JobStatus `protobuf:"bytes,4,rep,name=jobs,proto3" json:"jobs,omitempty"`
}

func (x *BlockStatus) Reset() {
	*x = BlockStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BlockStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlockStatus) ProtoMessage() {}

func (x *BlockStatus) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlockStatus.ProtoReflect.Descriptor instead.
func (*BlockStatus) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{8}
}

func (x *BlockStatus) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *BlockStatus) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_StatusUnknown
}

func (x *BlockStatus) GetHasError() bool {
	if x != nil {
		return x.HasError
	}
	return false
}

func (x *BlockStatus) GetJobs() []*JobStatus {
	if x != nil {
		return x.Jobs
	}
	return nil
}

// JobStatus holds the status of the Jobs.
type JobStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the Job called.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The description of the Job.
	Desc string `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
	// The args for the Job.
	Args map[string]string `protobuf:"bytes,3,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// The status of the Job.
	Status Status `protobuf:"varint,4,opt,name=status,proto3,enum=diskerase.Status" json:"status,omitempty"`
	// The error, if there was one.
	Error string `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *JobStatus) Reset() {
	*x = JobStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_diskerase_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JobStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JobStatus) ProtoMessage() {}

func (x *JobStatus) ProtoReflect() protoreflect.Message {
	mi := &file_diskerase_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JobStatus.ProtoReflect.Descriptor instead.
func (*JobStatus) Descriptor() ([]byte, []int) {
	return file_diskerase_proto_rawDescGZIP(), []int{9}
}

func (x *JobStatus) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *JobStatus) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *JobStatus) GetArgs() map[string]string {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *JobStatus) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_StatusUnknown
}

func (x *JobStatus) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

var File_diskerase_proto protoreflect.FileDescriptor

var file_diskerase_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x22, 0x5b, 0x0a, 0x07,
	0x57, 0x6f, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64,
	0x65, 0x73, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12,
	0x28, 0x0a, 0x06, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x10, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x63,
	0x6b, 0x52, 0x06, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x22, 0x1a, 0x0a, 0x08, 0x57, 0x6f, 0x72,
	0x6b, 0x52, 0x65, 0x73, 0x70, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x5e, 0x0a, 0x05, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x12,
	0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65,
	0x73, 0x63, 0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x61, 0x74, 0x65, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x72, 0x61, 0x74, 0x65, 0x4c, 0x69, 0x6d, 0x69,
	0x74, 0x12, 0x22, 0x0a, 0x04, 0x6a, 0x6f, 0x62, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x0e, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x4a, 0x6f, 0x62, 0x52,
	0x04, 0x6a, 0x6f, 0x62, 0x73, 0x22, 0x94, 0x01, 0x0a, 0x03, 0x4a, 0x6f, 0x62, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x2c, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e,
	0x4a, 0x6f, 0x62, 0x2e, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x61,
	0x72, 0x67, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x19, 0x0a, 0x07,
	0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x71, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x0a, 0x0a, 0x08, 0x45, 0x78, 0x65, 0x63, 0x52,
	0x65, 0x73, 0x70, 0x22, 0x1b, 0x0a, 0x09, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71,
	0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64,
	0x22, 0xd4, 0x01, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x29, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72,
	0x61, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x2e, 0x0a, 0x06, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x16, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x42,
	0x6c, 0x6f, 0x63, 0x6b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x68, 0x61, 0x64, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x68, 0x61, 0x64, 0x45, 0x72, 0x72, 0x6f, 0x72,
	0x73, 0x12, 0x24, 0x0a, 0x0e, 0x77, 0x61, 0x73, 0x5f, 0x65, 0x73, 0x5f, 0x73, 0x74, 0x6f, 0x70,
	0x70, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x77, 0x61, 0x73, 0x45, 0x73,
	0x53, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x22, 0x93, 0x01, 0x0a, 0x0b, 0x42, 0x6c, 0x6f, 0x63,
	0x6b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x29, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x64, 0x69,
	0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1b, 0x0a, 0x09, 0x68, 0x61, 0x73, 0x5f, 0x65, 0x72,
	0x72, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x68, 0x61, 0x73, 0x45, 0x72,
	0x72, 0x6f, 0x72, 0x12, 0x28, 0x0a, 0x04, 0x6a, 0x6f, 0x62, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x14, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x4a, 0x6f,
	0x62, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x04, 0x6a, 0x6f, 0x62, 0x73, 0x22, 0xe1, 0x01,
	0x0a, 0x09, 0x4a, 0x6f, 0x62, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64,
	0x65, 0x73, 0x63, 0x12, 0x32, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x1e, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x4a, 0x6f,
	0x62, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x12, 0x29, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72,
	0x61, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x1a, 0x37, 0x0a, 0x09, 0x41, 0x72, 0x67, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x2a, 0x6b, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x11, 0x0a, 0x0d, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x55, 0x6e, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x10, 0x00, 0x12, 0x14,
	0x0a, 0x10, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4e, 0x6f, 0x74, 0x53, 0x74, 0x61, 0x72, 0x74,
	0x65, 0x64, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x75,
	0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x10, 0x02, 0x12, 0x10, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x10, 0x03, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x43, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x10, 0x04, 0x32, 0xab,
	0x01, 0x0a, 0x08, 0x57, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x12, 0x33, 0x0a, 0x06, 0x53,
	0x75, 0x62, 0x6d, 0x69, 0x74, 0x12, 0x12, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73,
	0x65, 0x2e, 0x57, 0x6f, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x1a, 0x13, 0x2e, 0x64, 0x69, 0x73, 0x6b,
	0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x57, 0x6f, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x22, 0x00,
	0x12, 0x31, 0x0a, 0x04, 0x45, 0x78, 0x65, 0x63, 0x12, 0x12, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65,
	0x72, 0x61, 0x73, 0x65, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x71, 0x1a, 0x13, 0x2e, 0x64,
	0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x52, 0x65, 0x73,
	0x70, 0x22, 0x00, 0x12, 0x37, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x14, 0x2e,
	0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x65, 0x71, 0x1a, 0x15, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2e,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x22, 0x00, 0x42, 0x4f, 0x5a, 0x4d,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x50, 0x61, 0x63, 0x6b, 0x74,
	0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x69, 0x6e, 0x67, 0x2f, 0x47, 0x6f, 0x2d, 0x66, 0x6f,
	0x72, 0x2d, 0x44, 0x65, 0x76, 0x4f, 0x70, 0x73, 0x2f, 0x63, 0x68, 0x61, 0x70, 0x74, 0x65, 0x72,
	0x2f, 0x31, 0x38, 0x2f, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x64, 0x69, 0x73, 0x6b, 0x65, 0x72, 0x61, 0x73, 0x65, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_diskerase_proto_rawDescOnce sync.Once
	file_diskerase_proto_rawDescData = file_diskerase_proto_rawDesc
)

func file_diskerase_proto_rawDescGZIP() []byte {
	file_diskerase_proto_rawDescOnce.Do(func() {
		file_diskerase_proto_rawDescData = protoimpl.X.CompressGZIP(file_diskerase_proto_rawDescData)
	})
	return file_diskerase_proto_rawDescData
}

var file_diskerase_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_diskerase_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_diskerase_proto_goTypes = []interface{}{
	(Status)(0),         // 0: diskerase.Status
	(*WorkReq)(nil),     // 1: diskerase.WorkReq
	(*WorkResp)(nil),    // 2: diskerase.WorkResp
	(*Block)(nil),       // 3: diskerase.Block
	(*Job)(nil),         // 4: diskerase.Job
	(*ExecReq)(nil),     // 5: diskerase.ExecReq
	(*ExecResp)(nil),    // 6: diskerase.ExecResp
	(*StatusReq)(nil),   // 7: diskerase.StatusReq
	(*StatusResp)(nil),  // 8: diskerase.StatusResp
	(*BlockStatus)(nil), // 9: diskerase.BlockStatus
	(*JobStatus)(nil),   // 10: diskerase.JobStatus
	nil,                 // 11: diskerase.Job.ArgsEntry
	nil,                 // 12: diskerase.JobStatus.ArgsEntry
}
var file_diskerase_proto_depIdxs = []int32{
	3,  // 0: diskerase.WorkReq.blocks:type_name -> diskerase.Block
	4,  // 1: diskerase.Block.jobs:type_name -> diskerase.Job
	11, // 2: diskerase.Job.args:type_name -> diskerase.Job.ArgsEntry
	0,  // 3: diskerase.StatusResp.status:type_name -> diskerase.Status
	9,  // 4: diskerase.StatusResp.blocks:type_name -> diskerase.BlockStatus
	0,  // 5: diskerase.BlockStatus.status:type_name -> diskerase.Status
	10, // 6: diskerase.BlockStatus.jobs:type_name -> diskerase.JobStatus
	12, // 7: diskerase.JobStatus.args:type_name -> diskerase.JobStatus.ArgsEntry
	0,  // 8: diskerase.JobStatus.status:type_name -> diskerase.Status
	1,  // 9: diskerase.Workflow.Submit:input_type -> diskerase.WorkReq
	5,  // 10: diskerase.Workflow.Exec:input_type -> diskerase.ExecReq
	7,  // 11: diskerase.Workflow.Status:input_type -> diskerase.StatusReq
	2,  // 12: diskerase.Workflow.Submit:output_type -> diskerase.WorkResp
	6,  // 13: diskerase.Workflow.Exec:output_type -> diskerase.ExecResp
	8,  // 14: diskerase.Workflow.Status:output_type -> diskerase.StatusResp
	12, // [12:15] is the sub-list for method output_type
	9,  // [9:12] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_diskerase_proto_init() }
func file_diskerase_proto_init() {
	if File_diskerase_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_diskerase_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WorkReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WorkResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Block); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Job); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BlockStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_diskerase_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JobStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_diskerase_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_diskerase_proto_goTypes,
		DependencyIndexes: file_diskerase_proto_depIdxs,
		EnumInfos:         file_diskerase_proto_enumTypes,
		MessageInfos:      file_diskerase_proto_msgTypes,
	}.Build()
	File_diskerase_proto = out.File
	file_diskerase_proto_rawDesc = nil
	file_diskerase_proto_goTypes = nil
	file_diskerase_proto_depIdxs = nil
}
