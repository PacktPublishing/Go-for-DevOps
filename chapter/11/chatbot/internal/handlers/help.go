package handlers

var help = map[string]string{
	"list traces": `
list traces <opt1=val1 op2=val2>
Ex: list traces operation=AddPets() limit=5

list traces returns a list of Open Telemetry traces. Various
options are provided to allow for filtering what traces you see.

Options:
	operation
		Desc: Filter the traces that include this operation
		Ex: operation=server.AddPets()
	start
		Desc: Filter the trace by when in the past the trace started
		Ex: start=01/02/2021-15:04:05
	end:
		Desc: Filter the trace by when the trace ends
		Ex: end=01/02/2021-16:00:00
	limit:
		Desc: Limit the number of traces returned (default is 20)
		Ex: limit=5
	tags:
		Desc: Only include traces with these tags
		Ex: tags=[tag,tag2]
		Note: no spaces are allowed in the tag list
`,

	"show trace": `
show trace <trace id>
Ex: show trace 17b4f65b0d9f038e2a7bc5ea84309af2

show trace returns information about a particular Open Telemetry trace. 
This command has no options.
`,

	"change sampling": `
change sampling <type> <required value for type>
Ex: change sampling float .1

Sampling types:
	never
		Desc: Never sample unless another service or the RPC requests a trace
	always
		Desc: Sample very incoming RPC
	float
		Desc: Sample at a specific rate
		Required arg:
			<float>: Must be > 0 and <= 1
			Ex: change sampling float .1
`,

	"show logs": `
show logs <trace id>
Ex: show logs 17b4f65b0d9f038e2a7bc5ea84309af2

show logs returns all logs contained in a Open Telemetry trace. 
This command has no options.
`,
}
