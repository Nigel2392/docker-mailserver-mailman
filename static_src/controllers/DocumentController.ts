import { Controller } from "@hotwired/stimulus"
import { getCookie, setCookie } from "../utils/cookies";

export default class extends Controller {
    static targets: string[] = [
        "modeSwitch",
    ];

    static values = {
        cookieName: {
            type: String,
            default: "visual-mode",
        },
        modes: Array,
        navbarLinkSelector: String,
    }

    declare readonly hasModeSwitchTarget: boolean;
    declare readonly modeSwitchTarget: HTMLElement;

    declare readonly hasNavbarLinkSelectorValue: boolean;
    declare readonly navbarLinkSelectorValue: string;
    declare readonly hasCookieNameValue: boolean;
    declare readonly cookieNameValue: string;
    declare readonly modesValue: string[];

    
    declare currentMode: string;

    connect() {
        let currentMode = getCookie(this.cookieNameValue)
        if (currentMode == null || this.modesValue.indexOf(currentMode as string) == -1) {
            console.error("Invalid style mode cookie");
            this.currentMode = this.modesValue[0];
        } else {
            this.currentMode = currentMode as string;
        }

        this.bindFormValues();
        this.setupNavbarLinks();
    }

    nextStyleMode() {
        const idx = this.modesValue.indexOf(this.currentMode)
        if (idx + 1 < this.modesValue.length) {
            this.currentMode = this.modesValue[idx + 1]
        } else {
            this.currentMode = this.modesValue[0]
        }

        setCookie(this.cookieNameValue, this.currentMode, 365)
        location.reload();
    }

    private bindFormValues() {
        // Look for query parameters, if they match an input name; 
        // set the value of the input to the query parameter
        let inputs = document.querySelectorAll("input");
        let textareas = document.querySelectorAll("textarea");
        let all = [...inputs, ...textareas];
        let queryString = window.location.search;
        let urlParams = new URLSearchParams(queryString);
        for (let i = 0; i < all.length; i++) {
            let name = all[i].name;
            if (urlParams.has(name)) {
                all[i].value = all[i].value || urlParams.get(name) || "";
            }
        }
    }

    private setupNavbarLinks() {
        if (!this.hasNavbarLinkSelectorValue) {
            console.log("no navbarLinkSelector value specified")
            return
        }

        let urls = document.querySelectorAll(this.navbarLinkSelectorValue) as NodeListOf<HTMLAnchorElement>;
        if (!urls || !urls.length) {
            console.log(`no navbar links found for ${this.navbarLinkSelectorValue}`)
            return
        }

        let currentUrl = window.location.href;
        let path = currentUrl.split("#")[0].split("?")[0];
        for (let i = 0; i < urls.length; i++) {
            console.log("url: ", urls[i].href.split("#")[0], path)
            if (urls[i].href.split("#")[0] == path) {
                for (let j = 0; j < urls.length; j++) {
                    urls[j].classList.remove("active");
                }
                urls[i].classList.add("active");
            }
        }
    }
}

