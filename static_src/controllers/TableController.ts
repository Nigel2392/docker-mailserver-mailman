import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  static targets = [
      "selectAll", 
      "checkbox",
  ]

  declare readonly selectAllTarget: HTMLInputElement
  declare readonly hasSelectAllTarget: boolean
  declare readonly hasCheckboxTargets: boolean
  declare readonly checkboxTargets: HTMLInputElement[]

  get someChecked(): boolean {
      return this.hasCheckboxTargets && this.checkboxTargets.some(checkbox => checkbox.checked)
  }

  get allChecked(): boolean {
      return this.checkboxTargets.every(checkbox => checkbox.checked)
  }

  toggleAllCheckboxes(event: Event): void {
    const target = event.target as HTMLInputElement
    const isChecked = target.checked
    
    this.checkboxTargets.forEach(checkbox => {
      checkbox.checked = isChecked
    })
  }

  updateSelectAll(): void {
    if (this.hasSelectAllTarget) {
      this.selectAllTarget.checked = this.allChecked
    }
  }
}