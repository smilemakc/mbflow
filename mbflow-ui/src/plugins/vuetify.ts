/**
 * Vuetify plugin configuration
 */

import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

// Custom theme
const customTheme = {
    dark: false,
    colors: {
        primary: '#1976D2',
        secondary: '#424242',
        accent: '#82B1FF',
        error: '#FF5252',
        info: '#2196F3',
        success: '#4CAF50',
        warning: '#FB8C00',
        background: '#F5F5F5',
        surface: '#FFFFFF',
    },
}

export default createVuetify({
    components,
    directives,
    theme: {
        defaultTheme: 'customTheme',
        themes: {
            customTheme,
        },
    },
    defaults: {
        VBtn: {
            variant: 'flat',
            rounded: 'md',
        },
        VCard: {
            elevation: 2,
            rounded: 'lg',
        },
        VTextField: {
            variant: 'outlined',
            density: 'comfortable',
        },
        VSelect: {
            variant: 'outlined',
            density: 'comfortable',
        },
    },
})
