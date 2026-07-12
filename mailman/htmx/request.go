package htmx

import "net/http"

const (
	// Request headers
	HXBoosted               = "HX-Boosted"                 // indicates that the request is via an element using hx-boost
	HXCurrentURL            = "HX-Current-URL"             // the current URL of the browser
	HXHistoryRestoreRequest = "HX-History-Restore-Request" // “true” if the request is for history restoration after a miss in the local history cache
	HXPrompt                = "HX-Prompt"                  // the user response to an hx-prompt
	HXRequest               = "HX-Request"                 // always “true”
	HXTarget                = "HX-Target"                  // the id of the target element if it exists
	HXTriggerName           = "HX-Trigger-Name"            // the name of the triggered element if it exists

	// Both request and response headers
	HXTrigger = "HX-Trigger" // the id of the triggered element if it exists or allows you to trigger client-side events

	// Response headers
	HXLocation           = "HX-Location"             // allows you to do a client-side redirect that does not do a full page reload
	HXPushUrl            = "HX-Push-Url"             // pushes a new url into the history stack
	HXRedirect           = "HX-Redirect"             // can be used to do a client-side redirect to a new location
	HXRefresh            = "HX-Refresh"              // if set to “true” the client-side will do a full refresh of the page
	HXReplaceUrl         = "HX-Replace-Url"          // replaces the current URL in the location bar
	HXReswap             = "HX-Reswap"               // allows you to specify how the response will be swapped. See hx-swap for possible values
	HXRetarget           = "HX-Retarget"             // a CSS selector that updates the target of the content update to a different element on the page
	HXReselect           = "HX-Reselect"             // a CSS selector that allows you to choose which part of the response is used to be swapped in. Overrides an existing hx-select on the triggering element
	HXTriggerAfterSettle = "HX-Trigger-After-Settle" // allows you to trigger client-side events after the settle step
	HXTriggerAfterSwap   = "HX-Trigger-After-Swap"   // allows you to trigger client-side events after the swap step
)

func Is(h *http.Request) bool {
	return h.Header.Get(HXRequest) == "true"
}
func Boosted(h *http.Request) bool {
	return h.Header.Get(HXBoosted) == "true"
}
func CurrentURL(h *http.Request) string {
	return h.Header.Get(HXCurrentURL)
}
func HistoryRestoreRequest(h *http.Request) bool {
	return h.Header.Get(HXHistoryRestoreRequest) == "true"
}
func Prompt(h *http.Request) string {
	return h.Header.Get(HXPrompt)
}
func Target(h *http.Request) string {
	return h.Header.Get(HXTarget)
}
func TriggerName(h *http.Request) string {
	return h.Header.Get(HXTriggerName)
}
func TriggerID(h *http.Request) string {
	return h.Header.Get(HXTrigger)
}
