import { Controller } from "@hotwired/stimulus"


let messageAnimation = [
    { opacity: 1, height: "100%"},
    { opacity: 0, height: "0%"}
];

let messageAnimationOptions = {
    duration: 200,
    easing: 'ease-in-out'
}

export default class extends Controller {
    static targets: string[] = [
        "message",
    ];

    declare readonly hasMessageTarget: boolean;
    declare readonly messageTargets: HTMLElement[];

    connect() {
        (this.element as any).messages = this;
        setTimeout(this.timeout.bind(this), 4000)
    }

    private timeout() {
        for (let i = 0; i < this.messageTargets.length; i++) {
            const message = this.messageTargets[i]
            setTimeout(function(){

                // Gradually decrease the height of the message, such that the other messages go up
                let height = message.offsetHeight;
                let anim = message.animate([
                    { transform: "translateY(0px)", height: height + "px" },
                    { transform: "translateY(calc(-" + height + "px * 1.5))", height: "10px" }
                ], messageAnimationOptions)
                anim.onfinish = () => {
                    message.remove()
                }
            }, 1000 * i)
        }
    }

    addMessage(type: string, text: string): HTMLElement {
        const newMessage = document.createElement("div")
        newMessage.classList.add("message", `bg-${type}`)
        newMessage.dataset.action = "click->messages#closeMessage"

        const messageText = document.createElement("div")
        messageText.classList.add("message-text")
        messageText.innerText = text
        
        newMessage.appendChild(messageText)
        this.element.appendChild(newMessage)

        setTimeout(
            function() {
                let anim = newMessage.animate(messageAnimation, messageAnimationOptions)
                anim.onfinish = () => {
                    (newMessage as HTMLElement).remove()
                }
            },
            4000,
        )

        return newMessage
    }

    closeMessage(e: Event) {
        let target = e.target as HTMLElement;
        while (!target.classList.contains("message")) {
            target = target.parentElement as HTMLElement;
            if (!target) {
                console.error("target element not found...", e)
                break;
            }
        }
        let anim = (target as HTMLElement).animate(messageAnimation, messageAnimationOptions)
        anim.onfinish = () => {
            (target as HTMLElement).remove()
        }
    }
}

