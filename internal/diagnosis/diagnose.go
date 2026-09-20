package diagnosis

// This file is the entry point of the reasoning layer: one pure function from the results of one
// run to the findings and open questions of that run.
//
// Diagnose measures nothing. It reads the results it is handed and only those results: it does
// not dial, resolve, read a file, run a command or consult a seam, it has no way to obtain an
// observation the run did not report, and it never fills a gap. An observation that was not
// measured stays not measured and an attempt that produced nothing stays unresolved all the way
// into the conclusion that depends on it, because the rule that would state a confident answer
// needs a state the run did not produce and therefore cannot fire (R-HR-NF-03).

import "github.com/Luisalt20/herdr-reach/internal/probe"

// Diagnose reasons over one run's results and returns the diagnosis: one finding for every
// question the table answered and one open question for every question no rule matched, both in
// the table's declaration order.
//
// The results are read through Facts, so every fact is one observation's own state (DEV-2): a
// probe that reported several regions, or a binary beside its configuration, contributes one
// fact per observation, and no rule can match a probe's aggregate verdict where several
// observations were reported.
//
// A question no rule matched is an absence, not an error: it is reported as an open question
// naming the observable states the table needed. There is no fall-through default and no
// catch-all conclusion, so a conclusion exists only when a rule actually fired on the run's own
// measurements.
func Diagnose(results []probe.Result) Diagnosis {
	facts := Facts(results)

	var diagnosis Diagnosis
	for _, question := range questions() {
		rule, matched, ok := matchRule(question, facts)
		if !ok {
			diagnosis.OpenQuestions = append(diagnosis.OpenQuestions, OpenQuestion{
				Question:     question,
				NeededStates: neededStates(question),
			})
			continue
		}
		diagnosis.Findings = append(diagnosis.Findings, Finding{
			Question:   rule.Question,
			Rule:       rule.ID,
			Conclusion: rule.Conclusion.render(matched),
			// Copied, not aliased: the finding must not hand a caller a way to rewrite the table
			// row the conclusion came from.
			DependsOn: append([]string(nil), rule.DependsOn...),
			// Copied for the same reason: the matched facts are this pass's own values, and the
			// finding must not hand a caller a way to rewrite what it rests on.
			Evidence: append([]Fact(nil), matched...),
		})
	}
	return diagnosis
}
