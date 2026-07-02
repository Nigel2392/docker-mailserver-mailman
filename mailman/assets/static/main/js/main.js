document.body.addEventListener('htmx:responseError', function(event) {
    const xhr = event.detail.xhr;
    let errorMsg = xhr.responseText.trim();
    if (!errorMsg || errorMsg.includes("<html")) {
        errorMsg = `Server Error (${xhr.status})`;
    }

    addMessage("danger", errorMsg);
});

document.body.addEventListener('htmx:sendError', function(event) {
    addMessage("danger", "Network error: Could not connect to the server.");
});