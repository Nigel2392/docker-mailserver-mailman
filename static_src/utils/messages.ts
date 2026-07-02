function addMessage(type: string, text: string) {
    const m = document.querySelector(`[data-controller="messages"]`) as any;
    m.messages.addMessage(type, text)
}

export { addMessage };