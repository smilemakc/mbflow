# Form Components Library

Reusable form components for mbflow-react node configuration refactoring.

## Overview

This library provides a set of consistent, type-safe form components designed specifically for node configuration forms. All components follow the existing design patterns found in current node configs and integrate seamlessly with the VariableAutocomplete system.

## Components

### FormField
Wrapper component that provides label, hint, and error display.

```tsx
<FormField label="API Key" hint="Enter your key" required error={errors.apiKey}>
  <TextInput value={value} onChange={setValue} />
</FormField>
```

### TextInput
Text input with optional variable autocomplete support.

```tsx
// Regular input
<TextInput value={value} onChange={setValue} placeholder="Enter value" />

// With variables
<TextInput value={value} onChange={setValue} enableVariables />
```

### Select
Dropdown select with array or object options.

```tsx
// String array
<Select value={value} onChange={setValue} options={['opt1', 'opt2']} />

// With labels
<Select
  value={value}
  onChange={setValue}
  options={[
    { value: 'opt1', label: 'Option 1' },
    { value: 'opt2', label: 'Option 2' }
  ]}
/>
```

### NumberInput
Number input with min/max/step validation.

```tsx
<NumberInput
  value={temperature}
  onChange={setTemperature}
  min={0}
  max={2}
  step={0.1}
/>
```

### Textarea
Multi-line text input with optional monospace font and variables.

```tsx
// Regular
<Textarea value={value} onChange={setValue} rows={5} />

// With variables
<Textarea value={value} onChange={setValue} enableVariables rows={5} />

// Monospace (for code)
<Textarea value={value} onChange={setValue} monospace rows={8} />
```

### Checkbox
Checkbox with optional label.

```tsx
<Checkbox checked={value} onChange={setValue} label="Enable feature" />
```

## Styles

All components use centralized styles from `configStyles.ts`:

```tsx
import { configStyles } from '@/styles/configStyles';

// Input fields
configStyles.input
configStyles.select
configStyles.textarea
configStyles.textareaMonospace

// Labels and hints
configStyles.label
configStyles.labelRequired
configStyles.hint

// Sections
configStyles.section
configStyles.sectionTitle

// Checkboxes
configStyles.checkbox
configStyles.checkboxLabel

// Gradient headers
configStyles.gradientHeader.blue
configStyles.gradientHeader.green
configStyles.gradientHeader.amber
// ... etc

// Info boxes
configStyles.infoBox.info
configStyles.infoBox.warning
configStyles.infoBox.success
configStyles.infoBox.error
```

## File Structure

```
/Users/balashov/PycharmProjects/mbflow/mbflow-react/
├── styles/
│   └── configStyles.ts          # Centralized style definitions
└── components/
    └── ui/
        └── form/
            ├── FormField.tsx     # Wrapper with label/hint/error
            ├── TextInput.tsx     # Text input + variables
            ├── Select.tsx        # Dropdown select
            ├── Checkbox.tsx      # Checkbox with label
            ├── NumberInput.tsx   # Number input
            ├── Textarea.tsx      # Textarea + variables
            ├── index.ts          # Barrel export
            ├── README.md         # This file
            ├── USAGE_EXAMPLES.md # Detailed examples
            └── REFACTORING_GUIDE.md # Migration guide
```

## Usage

### Basic Import

```tsx
import { FormField, TextInput, Select, Checkbox, NumberInput, Textarea } from '@/components/ui/form';
import { configStyles } from '@/styles/configStyles';
```

### Complete Example

```tsx
import React, { useState } from 'react';
import { FormField, TextInput, Select, NumberInput, Checkbox, Textarea } from '@/components/ui/form';
import { configStyles } from '@/styles/configStyles';

export const MyNodeConfig: React.FC<Props> = ({ config, onChange }) => {
  const [localConfig, setLocalConfig] = useState(config);

  const updateConfig = (updates: Partial<typeof config>) => {
    const newConfig = { ...localConfig, ...updates };
    setLocalConfig(newConfig);
    onChange(newConfig);
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className={`${configStyles.gradientHeader.blue} rounded-lg p-4`}>
        <h3 className="font-semibold text-slate-900 dark:text-white text-sm">
          Configuration
        </h3>
        <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5">
          Configure your settings
        </p>
      </div>

      {/* Section */}
      <div className={configStyles.section}>
        <h4 className={configStyles.sectionTitle}>API Settings</h4>

        <FormField label="API Key" required>
          <TextInput
            value={localConfig.api_key}
            onChange={(api_key) => updateConfig({ api_key })}
            enableVariables
          />
        </FormField>

        <FormField label="Provider">
          <Select
            value={localConfig.provider}
            onChange={(provider) => updateConfig({ provider })}
            options={['openai', 'anthropic', 'google']}
          />
        </FormField>
      </div>

      {/* Fields */}
      <FormField label="Temperature" hint="0.0 to 2.0">
        <NumberInput
          value={localConfig.temperature}
          onChange={(temperature) => updateConfig({ temperature })}
          min={0}
          max={2}
          step={0.1}
        />
      </FormField>

      <FormField label="System Prompt">
        <Textarea
          value={localConfig.instruction}
          onChange={(instruction) => updateConfig({ instruction })}
          rows={5}
          enableVariables
        />
      </FormField>

      <Checkbox
        checked={localConfig.stream}
        onChange={(stream) => updateConfig({ stream })}
        label="Enable streaming"
      />
    </div>
  );
};
```

## Key Features

1. **Type Safety:** Full TypeScript support with proper type inference
2. **Variable Support:** Seamless integration with VariableAutocomplete
3. **Consistent Styling:** All components use centralized styles
4. **Dark Mode:** Automatic dark mode support
5. **Accessibility:** Built-in focus management and ARIA attributes
6. **Validation:** Error state support in FormField
7. **Flexibility:** Can override default className on any component

## Migration Benefits

- **25-40% less code** compared to old pattern
- **Consistent styling** across all forms
- **Easier maintenance** - update styles in one place
- **Better readability** - clearer component intent
- **Type safety** - reduced runtime errors
- **Faster development** - less boilerplate

## Documentation

- **USAGE_EXAMPLES.md** - Detailed usage examples for each component
- **REFACTORING_GUIDE.md** - Step-by-step guide to refactor existing configs
- **README.md** - This file (overview and quick reference)

## Next Steps

1. Use these components in new node configs
2. Gradually refactor existing node configs
3. Add more components as needed (e.g., RadioGroup, Switch)
4. Consider adding form validation utilities
5. Add unit tests for components

## Contributing

When adding new form patterns:
1. Extract common styles to `configStyles.ts`
2. Create reusable component in this directory
3. Add usage examples to `USAGE_EXAMPLES.md`
4. Update this README
5. Export from `index.ts`

## License

Part of the mbflow-react project.
