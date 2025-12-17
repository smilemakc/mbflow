/**
 * Form Components - Reusable form elements for node configurations
 *
 * These components provide a consistent interface for building node configuration
 * forms across the application. They follow the existing design patterns and
 * integrate seamlessly with the VariableAutocomplete system.
 *
 * Usage:
 * ```tsx
 * import { FormField, TextInput, Select, Checkbox, NumberInput, Textarea } from '@/components/ui/form';
 *
 * <FormField label="API Key" hint="Enter your API key" required>
 *   <TextInput value={apiKey} onChange={setApiKey} enableVariables />
 * </FormField>
 *
 * <FormField label="Method">
 *   <Select value={method} onChange={setMethod} options={['GET', 'POST', 'PUT']} />
 * </FormField>
 *
 * <FormField label="Temperature" hint="0.0 to 2.0">
 *   <NumberInput value={temp} onChange={setTemp} min={0} max={2} step={0.1} />
 * </FormField>
 *
 * <FormField label="Body">
 *   <Textarea value={body} onChange={setBody} monospace rows={8} />
 * </FormField>
 *
 * <Checkbox checked={enabled} onChange={setEnabled} label="Enable feature" />
 * ```
 */

export {FormField} from './FormField';
export {TextInput} from './TextInput';
export {Select} from './Select';
export {Checkbox} from './Checkbox';
export {NumberInput} from './NumberInput';
export {Textarea} from './Textarea';
