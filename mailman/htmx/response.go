package htmx

import (
	"net/http"
)

type HTMXResponse struct {
	http.ResponseWriter
}

func NewResponse(w http.ResponseWriter) *HTMXResponse {
	return &HTMXResponse{
		ResponseWriter: w,
	}
}

// allows you to trigger client-side events
func (h *HTMXResponse) Trigger(events string) {
	h.Header().Set(HXTrigger, events)
}

// allows you to do a client-side redirect that does not do a full page reload
func (h *HTMXResponse) Location(url string) {
	h.Header().Set(HXLocation, url)
}

// pushes a new url into the history stack
func (h *HTMXResponse) PushUrl(url string) {
	h.Header().Set(HXPushUrl, url)
}

// can be used to do a client-side redirect to a new location
func (h *HTMXResponse) Redirect(url string) {
	h.Header().Set(HXRedirect, url)
}

// if set to “true” the client-side will do a full refresh of the page
func (h *HTMXResponse) Refresh() {
	h.Header().Set(HXRefresh, "true")
}

// replaces the current URL in the location bar
func (h *HTMXResponse) ReplaceUrl(url string) {
	h.Header().Set(HXReplaceUrl, url)
}

// allows you to specify how the response will be swapped. See hx-swap for possible values
func (h *HTMXResponse) Reswap(swapStyle string) {
	h.Header().Set(HXReswap, swapStyle)
}

// a CSS selector that updates the target of the content update to a different element on the page
func (h *HTMXResponse) Retarget(selector string) {
	h.Header().Set(HXRetarget, selector)
}

// a CSS selector that allows you to choose which part of the response is used to be swapped in. Overrides an existing hx-select on the triggering element
func (h *HTMXResponse) Reselect(selector string) {
	h.Header().Set(HXReselect, selector)
}

// allows you to trigger client-side events after the settle step
func (h *HTMXResponse) TriggerAfterSettle(events string) {
	h.Header().Set(HXTriggerAfterSettle, events)
}

// allows you to trigger client-side events after the swap step
func (h *HTMXResponse) TriggerAfterSwap(events string) {
	h.Header().Set(HXTriggerAfterSwap, events)
}
