# Spike 06 — exam format and HCP Terraform free tier

Checked 2026-11-14 by reading the pages named below (no account created, nothing purchased). Each fact says where it came from; a fact that no HashiCorp page stated is marked as unconfirmed.

## Terraform Associate (004) exam

From HashiCorp's own pages ([certification page](https://developer.hashicorp.com/certifications/terraform-associate), [infrastructure automation certifications](https://developer.hashicorp.com/certifications/infrastructure-automation), [study guide](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-study-004), [exam content list](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-review-004)):

| Fact | Value | Source |
|---|---|---|
| Terraform version tested | 1.12 | all four pages |
| Duration | 1 hour | certification pages |
| Format | multiple choice | certification pages |
| Delivery | online proctored | certification page |
| Price | $70.50 USD plus local taxes and fees; "Free retake not included" | certification page |
| Credential validity | 2 years | certification page |
| Content | includes HCP Terraform; 004 adds objectives 4f, 4g, 4h and 8c compared with 003 | content list |
| Objectives | the same eight objectives and sub-objectives (1a to 8d) as in `ROADMAP.md` | content list |

**Not stated on any of those pages:** number of questions, passing score, the set of question types beyond "multiple choice".

**Unconfirmed (third-party sites only):** a web search returned many sites saying 57 questions and a 70 % pass mark; at least two of the sources say HashiCorp does not publish either figure and that 70 % is an estimate, and they disagree on the proctoring vendor. I did not find HashiCorp confirming them.

Meaning for tflings:
- `tflings exam` should default to 60 minutes and multiple choice, and make the question count a setting. Do not hard-code 57 or 70 % as official; label any default "unofficial".
- The ROADMAP item "verify the official exam format" is done except for question count and pass mark, which cannot be verified from official pages.

## HCP Terraform free plan

From the [IBM support page "HCP Terraform limits"](https://www.ibm.com/support/pages/hcp-terraform-limits) (modified 23 June 2026):

- Free plan covers **up to 500 managed resources** (resources under management) and **unlimited users** (the old five-member cap is gone).
- **One concurrent run.**
- Workspaces: no specific limit stated, subject to fair use.
- State retention on Free: at least the last 100 states across workspaces plus each workspace's latest state; older states kept six months.
- Not stated there: project limits, policy-as-code availability.

Conflicts and gaps:
- HashiCorp's [pricing page](https://www.hashicorp.com/en/pricing) as fetched shows no Free plan (cheapest listed is Essentials, from $0.10 per resource per month, plus a $500 trial credit) and has an FAQ "Is there still a free version of HCP Terraform?" whose answer was not in what I retrieved. So the free plan is confirmed by the IBM page but not by the pricing page.
- A third-party summary says the old free plan ended on 31 March 2026 and existing organisations moved to a 500-resource free tier. Single secondary source; unconfirmed.
- Not checked: whether signing up needs a payment card, regional availability, whether free organisations keep the `cloud` block and remote runs.

## Consequences for the HCP track (inference, not tested)

- 500 resources and one concurrent run are enough for a learner doing a few small exercises.
- A remote run executes in HashiCorp's cloud and cannot reach `localhost:4577`, so exercises that need Floci cannot use remote execution. Options to test before committing to the track: workspaces in local execution mode (state in HCP, run on the learner's machine), or hands-on exercises that use only `terraform_data` or `random`. The quiz mode covers objective 8 either way, as the ROADMAP already plans.
