package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/5/excel/visualization/internal/chart"

	"github.com/xuri/excelize/v2"
)

type CPUVendor string

const (
	UnknownCPUVendor CPUVendor = "Unknown"
	Intel            CPUVendor = "Intel"
	AMD              CPUVendor = "AMD"
)

var validCPUVendors = map[CPUVendor]bool{
	Intel: true,
	AMD:   true,
}

func main() {
	sheet, err := newServerSheet()
	if err != nil {
		panic(err)
	}

	sheet.add("svlaa01", 12, mustParse("10/27/2021"), Intel)
	sheet.add("svlac14", 13, mustParse("12/13/2021"), AMD)

	if err := sheet.render(); err != nil {
		panic(err)
	}
}

type summaries struct {
	cpuVendor cpuVendorSum
}

type cpuVendorSum struct {
	unknown, intel, amd int
}

type serverSheet struct {
	mu        sync.Mutex
	sheetName string
	xlsx      *excelize.File
	summaries *summaries
	nextRow   int
}

func newServerSheet() (*serverSheet, error) {
	s := &serverSheet{
		sheetName: "Sheet1",
		xlsx:      excelize.NewFile(),
		summaries: &summaries{},
		nextRow:   2,
	}

	s.xlsx.SetCellValue(s.sheetName, "A1", "Server Name")
	s.xlsx.SetCellValue(s.sheetName, "B1", "Generation")
	s.xlsx.SetCellValue(s.sheetName, "C1", "Acquisition Date")
	s.xlsx.SetCellValue(s.sheetName, "D1", "CPU Vendor")

	return s, nil
}

func (s *serverSheet) add(name string, gen int, acquisition time.Time, vendor CPUVendor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return errors.New("name cannot be blank")
	}
	if gen < 1 || gen > 13 {
		return errors.New("gen was not between 1 and including 13")
	}
	if acquisition.IsZero() {
		return errors.New("acquisition cannot be the zero time")
	}
	if !validCPUVendors[vendor] {
		return fmt.Errorf("vendor %v is not a valid vendor", vendor)
	}

	s.xlsx.SetCellValue(s.sheetName, "A"+strconv.Itoa(s.nextRow), name)
	s.xlsx.SetCellValue(s.sheetName, "B"+strconv.Itoa(s.nextRow), gen)
	s.xlsx.SetCellValue(s.sheetName, "C"+strconv.Itoa(s.nextRow), acquisition)
	s.xlsx.SetColWidth(s.sheetName, "C", "C", 20)
	s.xlsx.SetCellValue(s.sheetName, "D"+strconv.Itoa(s.nextRow), vendor)
	switch vendor {
	case Intel:
		s.summaries.cpuVendor.intel++
	case AMD:
		s.summaries.cpuVendor.amd++
	default:
		s.summaries.cpuVendor.unknown++
	}
	s.nextRow++

	return nil
}

func (s *serverSheet) render() error {
	s.writeSummaries()

	if err := s.createCPUChart(); err != nil {
		return fmt.Errorf("problem creating CPU chart: %w", err)
	}

	return s.xlsx.SaveAs("./Book1.xlsx")
}

func (s *serverSheet) writeSummaries() {
	s.xlsx.SetCellValue(s.sheetName, "F1", "Vendor Summary")
	s.xlsx.SetCellValue(s.sheetName, "F2", "Vendor")
	s.xlsx.SetCellValue(s.sheetName, "G2", "Total")

	s.xlsx.SetCellValue(s.sheetName, "F3", Intel)
	s.xlsx.SetCellValue(s.sheetName, "G3", s.summaries.cpuVendor.intel)
	s.xlsx.SetCellValue(s.sheetName, "F4", AMD)
	s.xlsx.SetCellValue(s.sheetName, "G4", s.summaries.cpuVendor.amd)
}

func (s *serverSheet) createCPUChart() error {
	c := chart.New()
	c.Type = "pie3D"
	c.Dimension = chart.FormatChartDimension{640, 480}
	c.Title = chart.FormatChartTitle{Name: "Server CPU Vendor Breakdown"}
	c.Format = chart.FormatPicture{
		FPrintsWithSheet: true,
		NoChangeAspect:   false,
		FLocksWithSheet:  false,
		OffsetX:          15,
		OffsetY:          10,
		XScale:           1.0,
		YScale:           1.0,
	}
	c.Legend = chart.FormatChartLegend{
		Position:      "bottom",
		ShowLegendKey: true,
	}
	c.Plotarea.ShowBubbleSize = true
	c.Plotarea.ShowCatName = true
	c.Plotarea.ShowLeaderLines = false
	c.Plotarea.ShowPercent = true
	c.Plotarea.ShowSerName = true
	c.ShowBlanksAs = "zero"

	c.Series = append(
		c.Series,
		chart.FormatChartSeries{
			Name:       `%s!$F$1`,
			Categories: fmt.Sprintf(`%s!$F$3:$F$4`, s.sheetName),
			Values:     fmt.Sprintf(`%s!$G$3:$G$4`, s.sheetName),
		},
	)

	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := s.xlsx.AddChart(s.sheetName, "I1", string(b)); err != nil {
		return err
	}

	return nil
}

func mustParse(s string) time.Time {
	const layout = "01/02/2006"

	t, err := time.Parse(layout, s)
	if err != nil {
		panic(err)
	}
	return t
}
