import { Controller } from "@hotwired/stimulus";
import tippy from 'tippy.js';
import 'tippy.js/dist/tippy.css';

export default class extends Controller<Element> {
    declare contentValue: string;
    declare placementValue: string;
    declare delayValue: number;
    declare durationValue: number;
    declare offsetValue: [number, number];
    static values = {
        content: String,
        placement: {
            type: String,
            default: "top",
        },
        delay: {
            type: Number,
            default: 0,
        },
        duration: {
            type: Number,
            default: 0,
        },
        offset: Array, 
    }
    declare tippyInstance: any;

    connect() {
        // 1. Start with the bare minimum
        const options: any = {
            content: this.contentValue,
        };

        // 2. Only pass properties if they actually exist. Otherwise, let Tippy handle defaults.
        if (this.placementValue) {
            options.placement = this.placementValue as any;
        }
        
        if (this.delayValue) {
            options.delay = this.delayValue;
        }
        
        if (this.durationValue) {
            options.duration = this.durationValue;
        }

        // 3. Stimulus defaults Arrays to `[]`. Don't pass empty arrays to Tippy.
        if (this.offsetValue && this.offsetValue.length === 2) {
            options.offset = this.offsetValue;
        }

        // 4. Initialize
        this.tippyInstance = tippy(this.element, options);
    }

    disconnect() {
        if (this.tippyInstance) {
            this.tippyInstance.destroy();
        }
    }

    contentValueChanged(value: string) {
        if (this.tippyInstance) {
            this.tippyInstance.setContent(value);
        }
    }
}