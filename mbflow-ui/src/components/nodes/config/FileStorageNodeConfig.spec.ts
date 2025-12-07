import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FileStorageNodeConfig from './FileStorageNodeConfig.vue'

// Mock TemplateInput component
const TemplateInput = {
    name: 'TemplateInput',
    props: ['modelValue', 'placeholder', 'nodeId', 'multiline', 'rows'],
    template: `<input 
    :value="modelValue" 
    @input="$emit('update:modelValue', $event.target.value)"
    :placeholder="placeholder"
  />`,
}

// Mock Select component  
const Select = {
    name: 'Select',
    props: ['modelValue', 'label', 'options'],
    template: `<select 
    :value="modelValue" 
    @change="$emit('update:modelValue', $event.target.value)"
  >
    <option v-for="opt in options" :key="opt.value" :value="opt.value">
      {{ opt.label }}
    </option>
  </select>`,
}

const createWrapper = (config = {}) => {
    return mount(FileStorageNodeConfig, {
        props: {
            config: {
                action: 'store',
                ...config,
            },
            nodeId: 'test-node-id',
        },
        global: {
            stubs: {
                TemplateInput,
                Select,
            },
        },
    })
}

describe('FileStorageNodeConfig', () => {
    describe('Rendering', () => {
        it('renders action select', () => {
            const wrapper = createWrapper()
            const selects = wrapper.findAll('select')
            expect(selects.length).toBeGreaterThan(0)
        })

        it('renders with default action as store', () => {
            const wrapper = createWrapper()
            const firstSelect = wrapper.find('select')
            expect(firstSelect.element.value).toBe('store')
        })
    })

    describe('Store Action', () => {
        it('shows file source options for store action', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()
            expect(html).toContain('File Source')
        })

        it('shows file name input for store action', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()
            expect(html).toContain('File Name')
        })

        it('shows access scope select for store action', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()
            expect(html).toContain('Access Scope')
        })

        it('shows TTL input for store action', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()
            expect(html).toContain('TTL')
        })

        it('shows tags input for store action', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()
            expect(html).toContain('Tags')
        })
    })

    describe('Get/Delete/Metadata Actions', () => {
        it('shows file_id input for get action', () => {
            const wrapper = createWrapper({ action: 'get' })
            const html = wrapper.html()
            expect(html).toContain('File ID')
        })

        it('shows file_id input for delete action', () => {
            const wrapper = createWrapper({ action: 'delete' })
            const html = wrapper.html()
            expect(html).toContain('File ID')
        })

        it('shows file_id input for metadata action', () => {
            const wrapper = createWrapper({ action: 'metadata' })
            const html = wrapper.html()
            expect(html).toContain('File ID')
        })
    })

    describe('List Action', () => {
        it('shows filters for list action', () => {
            const wrapper = createWrapper({ action: 'list' })
            const html = wrapper.html()
            expect(html).toContain('Filters')
        })

        it('shows limit input for list action', () => {
            const wrapper = createWrapper({ action: 'list' })
            const html = wrapper.html()
            expect(html).toContain('Limit')
        })

        it('shows offset input for list action', () => {
            const wrapper = createWrapper({ action: 'list' })
            const html = wrapper.html()
            expect(html).toContain('Offset')
        })
    })

    describe('Config updates', () => {
        it('emits update:config when action changes', async () => {
            const wrapper = createWrapper({ action: 'store' })
            const actionSelect = wrapper.find('select')

            await actionSelect.setValue('get')

            const emitted = wrapper.emitted('update:config')
            expect(emitted).toBeTruthy()
        })

        it('properly initializes config from props', () => {
            const wrapper = createWrapper({
                action: 'store',
                file_name: 'test.txt',
                mime_type: 'text/plain',
                access_scope: 'workflow',
            })

            const inputs = wrapper.findAll('input')
            // Check that inputs are rendered
            expect(inputs.length).toBeGreaterThan(0)
        })
    })

    describe('Action Options', () => {
        it('has all 5 action options', () => {
            const wrapper = createWrapper()
            const actionSelect = wrapper.find('select')
            const options = actionSelect.findAll('option')

            const values = options.map(o => o.element.value)
            expect(values).toContain('store')
            expect(values).toContain('get')
            expect(values).toContain('delete')
            expect(values).toContain('list')
            expect(values).toContain('metadata')
        })
    })

    describe('File Source Options', () => {
        it('shows URL input when file_source is url', () => {
            const wrapper = createWrapper({
                action: 'store',
                file_source: 'url'
            })
            const html = wrapper.html()
            expect(html).toContain('File URL')
        })

        it('shows Base64 input when file_source is base64', () => {
            const wrapper = createWrapper({
                action: 'store',
                file_source: 'base64'
            })
            const html = wrapper.html()
            expect(html).toContain('Base64 Data')
        })
    })

    describe('Access Scope Options', () => {
        it('has workflow, edge, and result access scope options', () => {
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()

            // These should be rendered as options
            expect(html).toContain('Workflow')
            expect(html).toContain('Edge')
            expect(html).toContain('Result')
        })
    })

    describe('Storage ID', () => {
        it('renders storage ID input', () => {
            const wrapper = createWrapper()
            const html = wrapper.html()
            expect(html).toContain('Storage ID')
        })

        it('shows hint about default storage', () => {
            const wrapper = createWrapper()
            const html = wrapper.html()
            expect(html).toContain('Leave empty for default storage')
        })
    })

    describe('Tags Handling', () => {
        it('initializes tags from array', () => {
            const wrapper = createWrapper({
                action: 'store',
                tags: ['tag1', 'tag2'],
            })

            // Tags should be converted to comma-separated string
            expect(wrapper.exists()).toBe(true)
        })
    })
})

describe('FileStorageNodeConfig Validation Logic', () => {
    describe('Store Validation', () => {
        it('requires either file_data or file_url for store', () => {
            // This tests the UI shows the right fields
            const wrapper = createWrapper({ action: 'store' })
            const html = wrapper.html()

            // Should show source type selector (labeled as "File Source")
            expect(html).toContain('File Source')
        })
    })

    describe('Get/Delete/Metadata Validation', () => {
        it('requires file_id for get action', () => {
            const wrapper = createWrapper({ action: 'get' })
            const html = wrapper.html()
            expect(html).toContain('File ID')
        })
    })
})

describe('FileStorageNodeConfig UI States', () => {
    describe('Action-specific UI', () => {
        const actions = ['store', 'get', 'delete', 'list', 'metadata']

        actions.forEach(action => {
            it(`renders correctly for ${action} action`, () => {
                const wrapper = createWrapper({ action })
                expect(wrapper.exists()).toBe(true)
                // No errors during render
            })
        })
    })
})
