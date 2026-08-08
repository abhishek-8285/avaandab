package digest

import (
	"fmt"
	"time"
)

type DailyDigestReport struct {
	Date            time.Time `json:"date"`
	Visitors        int       `json:"visitors"`
	NewSignups      int       `json:"new_signups"`
	Activated       int       `json:"activated"`
	Trials          int       `json:"trials"`
	Paid            int       `json:"paid"`
	MRR             string    `json:"mrr"`
	ChurnRiskCount  int       `json:"churn_risk_count"`
	CriticalErrors  int       `json:"critical_errors"`
	OpenIncidents   int       `json:"open_incidents"`
}

func FormatDailyDigest(report DailyDigestReport) string {
	return fmt.Sprintf(`📊 *FlyFleet Daily Report* (%s)

*Visitors*
%d

*New Signups*
%d

*Activated*
%d

*Trials*
%d

*Paid*
%d

*MRR*
%s

*Churn Risk*
%d Companies

*Critical Errors*
%d

*Open Incidents*
%d`, report.Date.Format("Jan 02, 2006"), report.Visitors, report.NewSignups, report.Activated, report.Trials, report.Paid, report.MRR, report.ChurnRiskCount, report.CriticalErrors, report.OpenIncidents)
}
