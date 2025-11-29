import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useNotificationStore = defineStore('notification', () => {
    const show = ref(false)
    const message = ref('')
    const color = ref('success')
    const timeout = ref(3000)

    function success(msg: string) {
        message.value = msg
        color.value = 'success'
        show.value = true
    }

    function error(msg: string) {
        message.value = msg
        color.value = 'error'
        show.value = true
    }

    function info(msg: string) {
        message.value = msg
        color.value = 'info'
        show.value = true
    }

    function warning(msg: string) {
        message.value = msg
        color.value = 'warning'
        show.value = true
    }

    return {
        show,
        message,
        color,
        timeout,
        success,
        error,
        info,
        warning,
    }
})
