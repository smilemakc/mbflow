import React, { useEffect, useMemo, useState } from 'react';
import { Save, Plus } from 'lucide-react';
import { Modal, Button } from '@/components/ui';
import { CopyButton } from '@/components/execution/CopyButton';
import { configStyles } from '@/styles/configStyles';
import { apiClient, getErrorMessage } from '@/lib/api';
import {
  buildCreateRequest,
  buildUpdateRequest,
  emptyTriggerForm,
  getWebhookUrl,
  hasErrors,
  triggerToForm,
  validateTriggerForm,
  type TriggerFormErrors,
  type TriggerFormValues,
} from '@/lib/triggers';
import { triggerService } from '@/services/triggerService';
import { useTranslation } from '@/store/translations';
import { TRIGGER_TYPES, type Trigger, type TriggerType } from '@/types/triggers';

export interface WorkflowOption {
  id: string;
  name: string;
}

export interface TriggerFormModalProps {
  isOpen: boolean;
  /** Trigger to edit; omit or null to create a new one. */
  trigger?: Trigger | null;
  workflows: WorkflowOption[];
  workflowsLoading?: boolean;
  /** Preselected workflow for new triggers. */
  defaultWorkflowId?: string;
  onClose: () => void;
  onSaved: (trigger: Trigger, mode: 'create' | 'edit') => void;
}

export const TriggerFormModal: React.FC<TriggerFormModalProps> = ({
  isOpen,
  trigger,
  workflows,
  workflowsLoading = false,
  defaultWorkflowId = '',
  onClose,
  onSaved,
}) => {
  const t = useTranslation();
  const tf = t.triggers.form;
  const isEdit = !!trigger;

  const [values, setValues] = useState<TriggerFormValues>(() =>
    trigger ? triggerToForm(trigger) : emptyTriggerForm(defaultWorkflowId),
  );
  const [errors, setErrors] = useState<TriggerFormErrors>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Reset the form whenever the dialog is (re)opened for a different trigger.
  useEffect(() => {
    if (!isOpen) return;
    setValues(trigger ? triggerToForm(trigger) : emptyTriggerForm(defaultWorkflowId));
    setErrors({});
    setSubmitError(null);
  }, [isOpen, trigger, defaultWorkflowId]);

  const typeLabels: Record<TriggerType, string> = {
    manual: t.triggers.manual,
    cron: t.triggers.cron,
    interval: t.triggers.interval,
    webhook: t.triggers.webhook,
    event: t.triggers.event,
  };

  const webhookUrl = useMemo(
    () =>
      trigger
        ? getWebhookUrl(trigger.id, apiClient.defaults.baseURL || '/api/v1', window.location.origin)
        : null,
    [trigger],
  );

  const set = <K extends keyof TriggerFormValues>(key: K, value: TriggerFormValues[K]) => {
    setValues((prev) => ({ ...prev, [key]: value }));
    if (errors[key]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
    }
  };

  const errorText = (key: keyof TriggerFormValues): string | null => {
    const code = errors[key];
    return code ? tf.errors[code] : null;
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    const validation = validateTriggerForm(values);
    setErrors(validation);
    if (hasErrors(validation)) return;

    setSubmitting(true);
    setSubmitError(null);
    try {
      const saved = trigger
        ? await triggerService.updateTrigger(trigger.id, buildUpdateRequest(values, trigger))
        : await triggerService.createTrigger(buildCreateRequest(values));
      onSaved(saved, trigger ? 'edit' : 'create');
    } catch (error) {
      setSubmitError(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const field = (
    key: keyof TriggerFormValues,
    label: string,
    control: React.ReactNode,
    hint?: string,
    required = false,
  ) => {
    const error = errorText(key);
    return (
      <div>
        <label htmlFor={`trigger-${key}`} className={configStyles.label}>
          {label}
          {required && <span className={configStyles.labelRequired}>*</span>}
        </label>
        {control}
        {error ? (
          <span role="alert" className="text-xs text-red-600 dark:text-red-400 mt-1 block">
            {error}
          </span>
        ) : (
          hint && <span className={configStyles.hint}>{hint}</span>
        )}
      </div>
    );
  };

  const inputClass = (key: keyof TriggerFormValues, base: string = configStyles.input) =>
    errors[key] ? `${base} border-red-400 dark:border-red-600` : base;

  const textInput = (key: keyof TriggerFormValues, placeholder?: string, type = 'text') => (
    <input
      id={`trigger-${key}`}
      type={type}
      value={values[key] as string}
      placeholder={placeholder}
      aria-invalid={!!errors[key]}
      onChange={(e) => set(key, e.target.value as never)}
      className={inputClass(key)}
      autoComplete="off"
    />
  );

  const jsonInput = (key: keyof TriggerFormValues, placeholder: string) => (
    <textarea
      id={`trigger-${key}`}
      rows={4}
      value={values[key] as string}
      placeholder={placeholder}
      aria-invalid={!!errors[key]}
      onChange={(e) => set(key, e.target.value as never)}
      className={inputClass(key, configStyles.textareaMonospace)}
    />
  );

  const footer = (
    <div className="flex justify-end gap-2">
      <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={submitting}>
        {tf.cancel}
      </Button>
      <Button
        type="submit"
        form="trigger-form"
        variant="primary"
        size="sm"
        loading={submitting}
        disabled={submitting}
        icon={isEdit ? <Save size={16} /> : <Plus size={16} />}
      >
        {isEdit ? tf.save : tf.create}
      </Button>
    </div>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEdit ? t.triggers.editModal.title : t.triggers.createModal.title}
      size="full"
      footer={footer}
      closeOnBackdrop={!submitting}
    >
      <form id="trigger-form" onSubmit={handleSubmit} noValidate className="space-y-4 max-h-[65vh] overflow-y-auto pr-1">
        {submitError && (
          <div
            role="alert"
            className="rounded-lg border border-red-200 dark:border-red-900/40 bg-red-50 dark:bg-red-900/20 px-3 py-2 text-sm text-red-700 dark:text-red-400"
          >
            {submitError}
          </div>
        )}

        {field(
          'workflowId',
          tf.workflow,
          <select
            id="trigger-workflowId"
            value={values.workflowId}
            disabled={isEdit || workflowsLoading}
            aria-invalid={!!errors.workflowId}
            onChange={(e) => set('workflowId', e.target.value)}
            className={inputClass('workflowId', configStyles.select)}
          >
            <option value="">{workflowsLoading ? tf.loadingWorkflows : tf.selectWorkflow}</option>
            {workflows.map((wf) => (
              <option key={wf.id} value={wf.id}>
                {wf.name}
              </option>
            ))}
            {values.workflowId && !workflows.some((wf) => wf.id === values.workflowId) && (
              <option value={values.workflowId}>{values.workflowId}</option>
            )}
          </select>,
          undefined,
          true,
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {field('name', tf.name, textInput('name', tf.namePlaceholder), undefined, true)}
          {field(
            'type',
            tf.type,
            <select
              id="trigger-type"
              value={values.type}
              onChange={(e) => set('type', e.target.value as TriggerType)}
              className={configStyles.select}
            >
              {TRIGGER_TYPES.map((type) => (
                <option key={type} value={type}>
                  {typeLabels[type]}
                </option>
              ))}
            </select>,
          )}
        </div>

        {field(
          'description',
          tf.description,
          <textarea
            id="trigger-description"
            rows={2}
            value={values.description}
            onChange={(e) => set('description', e.target.value)}
            className={configStyles.textarea}
          />,
        )}

        <div className={configStyles.section} data-testid={`trigger-config-${values.type}`}>
          {values.type === 'manual' && (
            <p className="text-sm text-slate-600 dark:text-slate-400">{tf.manualHint}</p>
          )}

          {values.type === 'cron' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {field('schedule', tf.schedule, textInput('schedule', '0 9 * * 1-5'), tf.scheduleHint, true)}
              {field('timezone', tf.timezone, textInput('timezone', 'UTC'), tf.timezoneHint)}
            </div>
          )}

          {values.type === 'interval' &&
            field('interval', tf.interval, textInput('interval', '5m'), tf.intervalHint, true)}

          {values.type === 'webhook' && (
            <>
              <div>
                <span className={configStyles.label}>{tf.webhookUrl}</span>
                {webhookUrl ? (
                  <div className="flex items-center gap-2">
                    <code
                      data-testid="webhook-url"
                      className="flex-1 truncate rounded bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 px-2 py-1.5 text-xs"
                    >
                      {webhookUrl}
                    </code>
                    <CopyButton text={webhookUrl} />
                  </div>
                ) : (
                  <span className={configStyles.hint}>{tf.webhookUrlHint}</span>
                )}
              </div>
              {field('secret', tf.secret, textInput('secret', '', 'password'), tf.secretHint)}
              {field(
                'ipWhitelist',
                tf.ipWhitelist,
                <textarea
                  id="trigger-ipWhitelist"
                  rows={2}
                  value={values.ipWhitelist}
                  placeholder="10.0.0.0/8, 203.0.113.7"
                  aria-invalid={!!errors.ipWhitelist}
                  onChange={(e) => set('ipWhitelist', e.target.value)}
                  className={inputClass('ipWhitelist', configStyles.textareaMonospace)}
                />,
                tf.ipWhitelistHint,
              )}
            </>
          )}

          {values.type === 'event' && (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {field('eventType', tf.eventType, textInput('eventType', tf.eventTypePlaceholder), undefined, true)}
                {field('eventSource', tf.eventSource, textInput('eventSource'))}
              </div>
              {field('eventFilter', tf.eventFilter, jsonInput('eventFilter', '{"plan": "pro"}'), tf.eventFilterHint)}
            </>
          )}
        </div>

        {field('input', tf.input, jsonInput('input', '{"key": "value"}'), tf.inputHint)}

        <label className="flex items-start gap-2 cursor-pointer">
          <input
            id="trigger-enabled"
            type="checkbox"
            checked={values.enabled}
            onChange={(e) => set('enabled', e.target.checked)}
            className={`${configStyles.checkbox} mt-0.5`}
          />
          <span>
            <span className={configStyles.checkboxLabel}>{tf.enabled}</span>
            <span className={configStyles.hint}>{tf.enabledHint}</span>
          </span>
        </label>
      </form>
    </Modal>
  );
};

export default TriggerFormModal;
