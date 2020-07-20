package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/segfaultax/go-nagios"
)

func runCheck(c *nagios.RangeCheck, pqlResult model.Value) {
	if pqlResult.Type().String() == "vector" {
		vec := pqlResult.(model.Vector)
		checkVector(c, vec)
	} else if pqlResult.Type().String() == "scalar" {
		vec := pqlResult.(*model.Scalar)
		val := float64(vec.Value)
		checkScalar(c, val)
	} else {
		unsupportedType := fmt.Errorf("return type should be either instant vector or scalar")
		c.Unknown("Error parsing Result: %v", unsupportedType)
		return
	}
}

func checkVector(c *nagios.RangeCheck, promVec model.Vector) {
	crit := c.Crit
	warn := c.Warn

	var oks model.Vector
	var crits model.Vector
	var warns model.Vector
	exit := nagios.StatusOK

	for _, r := range promVec {
		resultValue := float64(r.Value)
		c.AddPerfData(nagios.NewPerfData(r.Metric.String(), resultValue, ""))

		if crit.InRange(resultValue) {
			exit = nagios.StatusCrit
			crits = append(crits, r)
		} else if warn.InRange(resultValue) {
			if exit == nagios.StatusOK {
				exit = nagios.StatusWarn
			}
			warns = append(warns, r)
		} else {
			oks = append(oks, r)
		}
	}

	if len(crits) == 0 && len(warns) == 0 {
		c.OK("All metrics OK")
		return
	}
	var msgs []string

	for _, c := range crits {
		msgs = append(msgs, fmt.Sprintf("%s is critical (%0.02f)", c.Metric.String(), c.Value))
	}

	for _, w := range warns {
		msgs = append(msgs, fmt.Sprintf("%s is warning (%0.02f)", w.Metric.String(), w.Value))
	}

	c.Status = exit
	c.SetMessage(strings.Join(msgs, ", "))
}

func checkScalar(c *nagios.RangeCheck, scalarRes float64) {
	crit := c.Crit
	warn := c.Warn

	if crit.InRange(scalarRes) {
		c.Critical("%v is outside the critical thredshold", scalarRes)
	} else if warn.InRange(scalarRes) {
		c.Warning("%v is outside the warning thredshold", scalarRes)
	} else if math.IsNaN(scalarRes) {
		c.Unknown("NaN value returned")
	} else {
		c.OK("returned %v", scalarRes)
	}

	c.AddPerfData(nagios.NewPerfData("scalar", scalarRes, ""))
}
