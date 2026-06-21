import {
  Button,
  Stepper,
  StepperIndicator,
  StepperItem,
  StepperNav,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from '@e412/rnui-react';
import { Check } from 'lucide-react';
import type { ReactNode } from 'react';

export interface WizardStep {
  step: number;
  title: string;
}

export function WizardStepNav({
  steps,
  activeStep,
  completedSteps,
  onStepClick,
}: {
  steps: WizardStep[];
  activeStep: number;
  completedSteps: Set<number>;
  onStepClick: (step: number) => void;
}) {
  return (
    <Stepper value={activeStep} onValueChange={onStepClick}>
      <StepperNav>
        {steps.map((s, i) => (
          <StepperItem
            key={s.step}
            step={s.step}
            completed={completedSteps.has(s.step)}
            disabled={s.step > activeStep && !completedSteps.has(s.step)}
          >
            {/* type="button": rnui StepperTrigger renders a typeless <button>, which
                defaults to type="submit" inside a <form> and would submit on step click. */}
            <StepperTrigger type="button">
              <StepperIndicator>
                {completedSteps.has(s.step) ? <Check className="size-4" /> : s.step}
              </StepperIndicator>
              <StepperTitle className="hidden sm:block">{s.title}</StepperTitle>
            </StepperTrigger>
            {i < steps.length - 1 && <StepperSeparator />}
          </StepperItem>
        ))}
      </StepperNav>
    </Stepper>
  );
}

export function WizardActions({
  activeStep,
  lastStep,
  onBack,
  onNext,
  onCancel,
  onSubmit,
  submitting,
  submitLabel,
}: {
  activeStep: number;
  lastStep: number;
  onBack: () => void;
  onNext: () => void;
  onCancel: () => void;
  onSubmit: () => void;
  submitting: boolean;
  submitLabel: string;
}) {
  const isLast = activeStep === lastStep;
  return (
    <div className="flex items-center justify-between">
      <Button type="button" variant="ghost" onClick={onCancel} disabled={submitting}>
        Cancel
      </Button>
      <div className="flex items-center gap-2">
        {activeStep > 1 && (
          <Button type="button" variant="outline" onClick={onBack} disabled={submitting}>
            Back
          </Button>
        )}
        {/* type="button" + explicit onClick, NOT type="submit": the wizard must never
            hold a submit button (the Continue→submit morph reused its DOM node and
            auto-fired on the review step). Submission happens only on this click. */}
        {isLast ? (
          <Button type="button" onClick={onSubmit} disabled={submitting}>
            {submitting ? 'Saving…' : submitLabel}
          </Button>
        ) : (
          <Button type="button" onClick={onNext}>
            Continue
          </Button>
        )}
      </div>
    </div>
  );
}

export function ReviewSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium text-muted-foreground">{title}</h4>
      <dl className="divide-y rounded-lg border">{children}</dl>
    </div>
  );
}

export function ReviewRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
