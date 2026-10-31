# Recap video

`POST /v1/trips/{tripId}/recap/video` queues a render of the trip's recap as a 9:16 MP4 (VOY-42). `GET` on the same path returns the latest job for polling.

## Plan rules

Decided on the server when the job is made, through the entitlements service (`Recap`):

- Free: at most 30 seconds, with the Voyagami watermark.
- Plus, a Trip Pass, or Plus held by anyone in the crew: full length, no watermark.

Only owners and editors can start a render. Any member can read the job. A second request while a job is queued or rendering returns that job.

## Lifecycle

States: `queued`, `rendering`, `ready`, `failed`. `progress` is 0 to 100.

A scheduler calls `POST /v1/jobs/recap-videos` (bearer `JOBS_SECRET`, like the other jobs) every minute or two. Each run claims up to 3 jobs and renders them (`internal/recapvideo`):

- Hard time limit of 10 minutes per attempt.
- 3 attempts. A failed attempt goes back to the queue; the third failure ends the job as `failed` with a plain-English message.
- A job stuck in `rendering` for more than twice the time limit (the process died) is taken again.
- When a video is ready, the requester gets a push: "Your trip video is ready".
- Each attempt logs its job id, attempt number and wall time. That is the cost proxy until real render costs are known. No trip content is logged.

`url` on a ready job is a short-lived link made by the renderer's `URL` method.

## What is not built

There is **no renderer**. The lifecycle, queue, plan rules, retries and push are built and tested with a fake renderer, but nothing makes frames, picks music or encodes video. Until a `recapvideo.Renderer` is wired in `cmd/api/signin.go` (with `server.WithRecapVideo` and `server.WithRecapJob`), the routes answer 503, so the app should hide the button on a 503.

Still needed, and some of it needs Mihir:

- A renderer: ffmpeg in a Cloud Run job (or a similar worker), reading the recap facts and journal photos, writing to a storage bucket. Photos need the upload pipeline (VOY-161).
- A royalty-cleared music library (licensing decision).
- A Cloud Scheduler job for the route above.
- Pub/Sub instead of polling. The queue is a Firestore collection (`recapVideos`) and the scheduler polls it; move to Pub/Sub if latency matters.
- The share link page is out of scope for this ticket.
