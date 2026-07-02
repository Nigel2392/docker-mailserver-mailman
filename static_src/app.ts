import { Application} from "@hotwired/stimulus"
import TableController from "./controllers/TableController"
import TippyController from "./controllers/TippyController"
import FormModalController from "./controllers/FormModalController"
import DocumentController from "./controllers/DocumentController"
import MessagesController from "./controllers/MessagesController"
import { addMessage } from "./utils/messages"

window.Stimulus = Application.start()
window.Stimulus.register("document", DocumentController)
window.Stimulus.register("messages", MessagesController)
window.Stimulus.register("form-modal", FormModalController)
window.Stimulus.register("table-list", TableController)
window.Stimulus.register("tooltip", TippyController)

window.addMessage = addMessage