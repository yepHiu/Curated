import { createApp } from "vue"
import "vue-virtual-scroller/dist/vue-virtual-scroller.css"
import App from "./App.vue"
import router from "./router"
import "./style.css"

createApp(App).use(router).mount("#app")
