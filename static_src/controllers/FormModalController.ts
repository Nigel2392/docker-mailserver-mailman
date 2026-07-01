import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
    static targets = [
        "underlay", 
        "modal",
        "form",
    ];

    declare readonly hasUnderlayTarget: boolean;
    declare readonly hasModalTargets: boolean;
    declare readonly underlayTarget: HTMLElement;
    declare readonly modalTarget: HTMLElement;
    declare readonly formTarget: HTMLFormElement;

    connect() {
    }

    close() {
        this.element.remove()
    }
}