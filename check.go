package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/segfaultax/go-nagios"
)

func checkVector(c *nagios.Check, promVec model.Vector, critThreshold string, warnThreshold string) {
	crit, _ := nagios.ParseRange(critThreshold)
	warn, _ := nagios.ParseRange(warnThreshold)

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
