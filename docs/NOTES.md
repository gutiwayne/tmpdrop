# Implementation notes

The code carries no comments. This file holds the decisions that are not
obvious from reading it — mostly things that were verified against a real
ConvertX instance and a real browser, and that will break if someone
"simplifies" them.

## Downloads

**A ranged request or a `HEAD` must never count as a download.** Go's `ServeMux`
routes `HEAD` to the `GET` pattern, so both reach the download handler. Counting
them would spend a `max_downloads: 1` link before the person ever clicked it —
any chat client that unfurls the URL would burn it.

`http.ServeContent` is called with a **zero modtime on purpose**. With a real
timestamp it answers conditional requests with `304`, which sends no body but
still falls through to the counter.

The counter runs *after* the body is served, not before. When the last allowed
download is delivered the file is deleted immediately rather than left to expire.

## Uploads

Uploads are read with `MultipartReader` and streamed to a temp file inside the
storage directory. Nothing is buffered in memory, so an 80 MB upload costs about
2 MB of RAM. The earlier `ParseMultipartForm` version held the entire file in
memory.

Because of this the form fields may arrive in **any order**: the file part is
spooled first and the metadata applied afterwards.

`Add` also goes through a temp file and an atomic rename. Writing straight into
the blob directory would let a concurrent sweep delete the half-written file,
since the sweeper removes blobs the manifest does not reference.

## Storage

The manifest is written temp-first and renamed, under a mutex. `Sweep` removes
expired entries, blobs with no manifest entry, and abandoned upload temporaries.

Reclamation has two guards, both of which exist because their absence destroyed
real files during development:

- **A blob younger than an hour is never reclaimed.** Without that, a sweep
  running while an upload is in flight deletes the file being written.
- **Nothing is reclaimed at all when the manifest lists no files but blobs are
  on disk.** That combination means the manifest was lost, not that every blob
  is garbage; deleting them is exactly the wrong response. The store flags the
  condition through `ReclaimBlocked` and the sweeper logs a warning every tick
  so it cannot pass unnoticed.

## Security

`X-Forwarded-For` is believed **only** when the request arrives from an address
listed in `trusted_proxies`. Otherwise anyone could forge a header and walk
around the per-client quota and the password rate limit.

Verifying a password is deliberately expensive (PBKDF2, 600k iterations), so
failed attempts are rate limited per client. Without that, guessing is a cheap
way to burn the server's CPU.

Files are served with `Content-Security-Policy: default-src 'none'; sandbox`,
and HTML and SVG are always sent as attachments even when `?inline=1` is asked
for. Uploaded content is never trusted with the application's own origin.

## Fonts

`mime.AddExtensionType(".woff2", "font/woff2")` is required. Go's built-in table
has no entry for woff2 and the runtime image ships no `/etc/mime.types`, so
without it the fonts are served as `application/octet-stream`.

## Conversion

### Remote conversion (ConvertX)

The converter tables are tuned to what a real ConvertX 0.18 instance can
actually do. Every format offered in the UI was tested end to end. Three
exclusions are deliberate:

- **dasel is absent entirely.** ConvertX invokes it as `dasel --file`, a flag
  dasel v2 removed, so every job through it fails. Data formats (json, yaml,
  toml) are therefore fed to LibreOffice as plain text instead.
- **No avif, heic or heif from vips.** The vips build ConvertX ships has no HEIF
  encoder and fails with `heifsave: Unsupported compression`.
- **Audio sources are never offered gif or apng**, and only tabular sources are
  offered spreadsheets. Both combinations fail in ConvertX for obvious reasons.

Plain-text sources — scripts, config and data files — are mapped to `txt` so
LibreOffice will accept them. ConvertX takes the original extension and treats
the content as text, so no renaming is needed on upload.

A finished conversion stays queryable for ten minutes; only a *running* one
blocks a new conversion of the same file. Dropping finished jobs immediately
would return 404 to a poll that is still in flight.

**Nothing is buffered in memory here either, and that took work.** The obvious
implementation reads the blob with `os.ReadFile`, copies it again into a
`bytes.Buffer` for the multipart body and reads the result with `io.ReadAll` —
three copies, so converting a 100 MB file needs a container with more than
300 MB of RAM. Instead the source is piped straight from disk into the multipart
writer and the result is streamed into a store temp file, the same
`TempFile` → `Adopt` path an upload takes.

`Convert` takes a `Source` with an `Open` function rather than an `io.Reader`,
because a session that expires mid-conversion makes it retry the whole exchange,
and the second attempt has to read the file from the beginning again. For the
same reason the retry is only ever triggered by a `401`, which the download
checks *before* copying any bytes — a retry can never append to a result that
was already half written.

The multipart body is sent from an `io.Pipe`, whose length nothing knows in
advance, so `Content-Length` is computed with `multipartLen`: it runs a real
`multipart.Writer` over a counting writer with the same boundary and adds the
file size. Assembling that number by hand from the boundary and header strings
works right up until one of them changes. Sending it chunked instead would be
simpler, but leaves the upload at the mercy of how the other end handles a
multipart request with no length.

## Registration

`ensureAuth` logs in, and registers only if the login failed. A real ConvertX
answers an unknown account with **403**, and 403 is what makes the client try to
register — but it treats a `401` the same way on purpose. The two are separate
sentinel errors internally, and an earlier version only reacted to one of them,
which meant auto-registration would have silently stopped working if ConvertX
ever changed that status. There is a test for the 401 path.

Registration failures keep the response body (bounded) in the error, because
"already in use" is how ConvertX reports an account that exists with a different
password, and `ensureAuth` reads that text to tell it apart from a real failure.

### Local conversion (gpsbabel)

As of v2.3.0, GPS and sports formats (KMZ, KML, FIT, TCX, OZI, GDB, GeoJSON)
are converted **locally using gpsbabel**, without ConvertX overhead or network
round-trips. The binary ships with Ubuntu 24.04; Alpine 3.20 was dropped because
gpsbabel 1.9+ requires Qt6 libraries that conflict with the distro's older Qt5
stack.

**Format translation:** User-facing extensions (e.g., `.tcx`) map to gpsbabel
codes (e.g., `gtrnctr`). Mappings live in `localConvertGpsbabel()` in
`internal/convertx/client.go`. If a new format needs different input or output
codes, update that switch statement.

**Converter selection:** API clients may omit `converter`. It is resolved with
`ConverterFor()` before anything else, so a GPS conversion always takes the
local path; sending it to ConvertX would leave the job hanging until the
deadline, because ConvertX does not know these formats.

**KMZ handling:** KMZ is a zip archive containing KML. The archive is spooled to
a temporary file (zip needs random access) and the first `.kml` inside is
streamed to gpsbabel's input file, capped at 512 MiB. Nothing is held in memory,
so a large KMZ or a zip bomb cannot exhaust a small container memory limit.

**MIME type detection:** `detectType()` in `internal/server/handlers.go` types
both uploads and conversion results. On top of `http.DetectContentType` it
recognises geospatial formats by content (`<gpx`, `<kml`,
`<TrainingCenterDatabase`, a GeoJSON object with `"geometry"`) and FIT by its
header (header size 12 or 14 in byte 0, `.FIT` in bytes 8-11). None of these
types is inline-safe, so they are always served as downloads.

## Metadata stripping

As of v2.4.0, uploaded images automatically have their EXIF and GPS metadata stripped **by default**, via `exiftool`. Since there is no authentication, preventing accidental metadata leakage is a privacy baseline.

The command uses `-all=` to remove all tags, combined with `-tagsfromfile @ -Orientation` to preserve the `Orientation` EXIF tag. This tag is critical for mobile phone photos: many are stored as landscape at the pixel level and rely on the Orientation tag to tell the viewer to rotate them. Removing it would make all such photos display sideways without changing the actual image data.

Invocation happens in `handleUpload` after form parsing and before `store.Adopt`, so the clean file is persisted. If exiftool fails, a warning is logged but the upload continues — metadata stripping is best-effort, not a blocker.

The feature is **opt-in from the UI** (checkbox "Strip GPS/EXIF") but **ON by default**, so older API clients and basic curl uploads are protected even if they don't send the field.

Applies only to `image/*` MIME types. Videos and other formats are left untouched; exiftool on multi-gigabyte files is slow and would risk timeouts.
