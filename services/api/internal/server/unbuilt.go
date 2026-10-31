package server

// msgNotBuilt is the answer of a route whose ticket has not landed. The routes
// live in per-ticket unbuilt_<ticket>.go files; a ticket deletes its file, and
// the last one deletes this file too.
const msgNotBuilt = "This is not available yet."
