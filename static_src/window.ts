import { Application, Controller } from "@hotwired/stimulus";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        StimulusController: typeof Controller;
        i18n: {
            gettext(str: string, ...args: any): string,
            ngettext(singular: string, plural: string, n: any, ...args: any): string,
        };
    }
}