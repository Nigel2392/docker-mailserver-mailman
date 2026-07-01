import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
    static targets = [
        "selectAll", 
        "checkbox",
        "actionButton",
    ]

    declare readonly hasSelectAllTarget: boolean;
    declare readonly selectAllTarget: HTMLInputElement;
    declare readonly checkboxTargets: HTMLInputElement[];
    declare readonly actionButtonTargets: HTMLElement[];

    get someChecked(): boolean {
        return this.checkboxTargets.some(checkbox => checkbox.checked);
    }

    get allChecked(): boolean {
        return this.checkboxTargets.length > 0 && this.checkboxTargets.every(checkbox => checkbox.checked);
    }

    connect(): void {
        this.updateSelectAll();
    }

    toggleAllCheckboxes(event: Event): void {
        const target = event.target as HTMLInputElement;
        const isChecked = target.checked;

        this.checkboxTargets.forEach(checkbox => {
            checkbox.checked = isChecked;
        });
        
        this.updateSelectAll();
    }

    updateSelectAll(): void {
        // 1. Calculate how many are currently checked
        const checkedCount = this.checkboxTargets.filter(c => c.checked).length;

        if (this.hasSelectAllTarget) {
            this.selectAllTarget.checked = (checkedCount === this.checkboxTargets.length && checkedCount > 0);
        }

        this.actionButtonTargets.forEach(button => {
            const min = parseInt(button.dataset.minChecked || "0", 10);
            
            const max = button.dataset.maxChecked 
                ? parseInt(button.dataset.maxChecked, 10) 
                : Infinity;

            // Determine if button requirements are met
            const requirementsMet = checkedCount >= min && checkedCount <= max;
            if (!requirementsMet) {
                button.classList.add("disabled");
                (button as any).disabled = true
            } else {
                button.classList.remove("disabled");
                (button as any).disabled = false
            }
        });
    }
}