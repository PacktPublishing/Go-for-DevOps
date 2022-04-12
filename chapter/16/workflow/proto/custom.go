package diskerase

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rodaine/table"
)

// CLISummary() provides the StatusResp in a summary format that is useful for
// viewing in a CLI application. It summarizes all blocks into single lines except
// for the block that is currently running.
func (x *StatusResp) CLISummary(id string) string {
	if len(x.Blocks) == 0 {
		return "no blocks defined"
	}

	blockTitle := color.New(color.FgCyan).Add(color.Underline)
	name := color.New(color.FgGreen)
	desc := color.New(color.FgYellow)

	buff := strings.Builder{}
	buff.WriteString(fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC1123)))
	buff.WriteString(fmt.Sprintf("Workflow: %s\n", id))
	name.Fprintln(&buff, "Name: "+x.Name)
	desc.Fprintln(&buff, "Description: "+x.Desc)

	if i, block := x.findRunning(x.Blocks); i != -1 {
		blockTitle.Fprintln(&buff, fmt.Sprintf("\nRunning Block(%d): %s", i, block.Desc))
		x.writeRunning(&buff, block)
	}

	blockTitle.Fprintln(&buff, "\nBlock Summaries")
	x.writeOthers(&buff, x.Blocks)

	return buff.String()
}

func (x *StatusResp) writeOthers(buff *strings.Builder, blocks []*BlockStatus) {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Block Number", "Desc", "Status").WithWriter(buff)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for i, block := range blocks {
		tbl.AddRow(i, block.Desc, block.Status)
		if block.Status == Status_StatusRunning {
			continue
		}
	}
	tbl.Print()
}

func (x *StatusResp) findRunning(blocks []*BlockStatus) (int, *BlockStatus) {
	for i, b := range blocks {
		if b.Status == Status_StatusRunning {
			return i, b
		}
	}
	return -1, nil
}

func (x *StatusResp) writeRunning(buff *strings.Builder, block *BlockStatus) {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Job Number", "Desc", "Status").WithWriter(buff)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for i, job := range block.Jobs {
		tbl.AddRow(i, job.Desc, job.Status)
	}
	tbl.Print()
	return
}
