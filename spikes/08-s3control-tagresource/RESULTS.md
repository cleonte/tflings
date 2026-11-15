# Spike 08: S3 Control TagResource replaces the tag set (Floci 2.4.0)

Date: 2026-11-15. Floci 2.4.0 (image digest in CLAUDE.md), AWS provider 6.73.0, Terraform 1.12.2.

## What happens

Terraform changes the tags of an `aws_s3_bucket` from `{a=1}` to `{a=1, b=2, c=3}`. The apply says it
added `b` and `c`. Afterwards the state, and Floci, hold only `{b=2, c=3}`: `a` is gone. The next
plan wants to add `a` again, so the plan is never empty. (Terraform with `-var 'tags={a="1"}'`, then
`-var 'tags={a="1",b="2",c="3"}'`; run by hand, same result every time.)

## Cause

For a change of tags the provider calls S3 Control `TagResource` with only the **added** tags, and
`UntagResource` for removed ones (`POST /v20180820/tags/arn:aws:s3:::BUCKET`, seen in the provider's
debug log). Floci's `TagResource` **replaces** the whole tag set with the tags in the request.
`repro.sh` shows it with `curl` alone, no Terraform:

```
== tag a=1                      -> ListTagsForResource: a=1
== tag b=2 and c=3              -> ListTagsForResource: b=2 c=3      (a is lost)
```

## What AWS does (not run here)

The AWS documentation describes S3 Control `TagResource` as creating new tags or updating existing
ones, so the tags that are not in the request stay. I did not call real AWS (tflings never does),
so this comes from the documentation, not from a test.

## Not checked

- Floci 2.5.0 (released 2026-11-15): not tried; the container here is 2.4.0.
- No existing issue was found in floci-io/floci with the searches "TagResource s3control tags merge"
  and "s3 bucket tags" (2026-11-15).

## Effect on tflings

No exercise changes the tags of a bucket (checked with grep over `exercises/` and `solutions/`), and
exercises that add tags use `aws_vpc`, so nothing in the course depends on this.

## Issue text (not posted)

> **S3 Control TagResource replaces all tags instead of adding to them**
>
> Floci 2.4.0. `POST /v20180820/tags/{arn}` (S3 Control `TagResource`) with tags `b`, `c` on a bucket
> that already has tag `a` leaves only `b` and `c`. `TagResource` should add or update the given tags
> and leave the others (`PutBucketTagging` on the S3 API is the call that replaces the set).
> The AWS Terraform provider (6.x) uses `TagResource` for the tags it adds, so a change from
> `{a=1}` to `{a=1,b=2,c=3}` ends with `{b=2,c=3}`, and every later plan shows the diff again.
>
> Repro, no Terraform needed (`repro.sh` in tflings, spikes/08-s3control-tagresource): create a bucket,
> `TagResource` with `a=1`, then `TagResource` with `b=2` and `c=3`, then `ListTagsForResource`.
