package exercises

import (
	"fmt"
	"slices"
)

// Availability is what this machine can run: whether Terraform is there,
// whether the tflings Floci container answers, and which optional services are
// set up.
type Availability struct {
	Terraform bool
	Floci     bool
	HCP       bool
}

// SkipReason says why the exercise cannot run on this machine, or is empty if
// it can. A quiz always runs.
func (e Exercise) SkipReason(av Availability) string {
	switch {
	case e.NeedsTerraform() && !av.Terraform:
		return "needs Terraform: run `tflings doctor`"
	case e.NeedsFloci() && !av.Floci:
		return "needs Floci: run `tflings aws up`"
	case slices.Contains(e.Requires, RequiresHCP) && !av.HCP:
		return "needs an HCP Terraform token: set TF_TOKEN_app_terraform_io (this exercise is optional)"
	}
	return ""
}

// Progress counts the exercises that are done and the exercises there are.
func Progress(infoFile string) (done, total int, err error) {
	all, err := List(infoFile)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range all {
		if e.State() == Done {
			done++
		}
	}
	return done, len(all), nil
}

// Skipped returns the exercises that cannot run here.
func Skipped(infoFile string, av Availability) ([]Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return nil, err
	}
	var skipped []Exercise
	for _, e := range all {
		if e.SkipReason(av) != "" {
			skipped = append(skipped, e)
		}
	}
	return skipped, nil
}

// TopicProgress is how far the learner is in one topic.
type TopicProgress struct {
	Topic       string
	Done, Total int
}

// ProgressByTopic returns the progress of every topic, in manifest order.
func ProgressByTopic(infoFile string) ([]TopicProgress, error) {
	all, err := List(infoFile)
	if err != nil {
		return nil, err
	}
	var out []TopicProgress
	for _, e := range all {
		if len(out) == 0 || out[len(out)-1].Topic != e.Topic {
			out = append(out, TopicProgress{Topic: e.Topic})
		}
		last := &out[len(out)-1]
		last.Total++
		if e.State() == Done {
			last.Done++
		}
	}
	return out, nil
}

// Describe is the one-line form of an exercise for listings.
func (e Exercise) Describe() string {
	if e.Objective != "" {
		return fmt.Sprintf("%s (%s, objective %s)", e.Name, e.Mode, e.Objective)
	}
	return fmt.Sprintf("%s (%s)", e.Name, e.Mode)
}
