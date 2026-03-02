import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardHeading,
  CardTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Skeleton,
  Spinner,
  Textarea,
} from '@e412/titanium';
import { Eye, Lock, Mail, RotateCcw } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';

import { useACLBranding, useUpdateACLBranding } from '@/hooks';
import { sanitizeCSS } from '@/lib/css-sanitizer';
import type { ACLBranding } from '@/types/acl';

interface BrandingFormValues {
  logo_url: string;
  primary_color: string;
  background_color: string;
  title: string;
  subtitle: string;
  footer_text: string;
  custom_css: string;
}

const DEFAULT_BRANDING: BrandingFormValues = {
  logo_url: '',
  primary_color: '#3b82f6',
  background_color: '#ffffff',
  title: 'Login Required',
  subtitle: 'Sign in to continue',
  footer_text: '',
  custom_css: '',
};

function brandingToForm(branding: ACLBranding | undefined): BrandingFormValues {
  if (!branding) return DEFAULT_BRANDING;
  return {
    logo_url: branding.logo_url ?? '',
    primary_color: branding.primary_color ?? DEFAULT_BRANDING.primary_color,
    background_color: branding.background_color ?? DEFAULT_BRANDING.background_color,
    title: branding.title ?? DEFAULT_BRANDING.title,
    subtitle: branding.subtitle ?? '',
    footer_text: branding.footer_text ?? '',
    custom_css: branding.custom_css ?? '',
  };
}

// Color input with hex validation
function ColorInput({
  value,
  onChange,
  id,
  label,
}: {
  value: string;
  onChange: (value: string) => void;
  id: string;
  label: string;
}) {
  const [inputValue, setInputValue] = useState(value);
  const isValidHex = /^#[0-9A-Fa-f]{6}$/.test(inputValue);

  useEffect(() => {
    setInputValue(value);
  }, [value]);

  const handleInputChange = (newValue: string) => {
    setInputValue(newValue);
    if (/^#[0-9A-Fa-f]{6}$/.test(newValue)) {
      onChange(newValue);
    }
  };

  const handleColorPickerChange = (newValue: string) => {
    setInputValue(newValue);
    onChange(newValue);
  };

  return (
    <Field data-invalid={!isValidHex && inputValue.length > 0}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <FieldContent>
        <div className="flex gap-2">
          <Input
            id={id}
            value={inputValue}
            onChange={(e) => handleInputChange(e.target.value)}
            placeholder="#000000"
            className="flex-1 font-mono"
            aria-invalid={!isValidHex && inputValue.length > 0}
          />
          <div className="relative">
            <input
              type="color"
              value={isValidHex ? inputValue : '#000000'}
              onChange={(e) => handleColorPickerChange(e.target.value)}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              aria-label={`Color picker for ${label}`}
            />
            <div
              className="w-10 h-10 rounded-md border border-input cursor-pointer shadow-sm"
              style={{ backgroundColor: isValidHex ? inputValue : '#ffffff' }}
            />
          </div>
        </div>
      </FieldContent>
      {!isValidHex && inputValue.length > 0 && (
        <FieldError errors={[{ message: 'Please enter a valid hex color (e.g., #3b82f6)' }]} />
      )}
    </Field>
  );
}

// Login page preview component
function LoginPreview({
  logoUrl,
  primaryColor,
  backgroundColor,
  title,
  subtitle,
  footerText,
  customCss,
}: {
  logoUrl: string;
  primaryColor: string;
  backgroundColor: string;
  title: string;
  subtitle: string;
  footerText: string;
  customCss: string;
}) {
  const [imageError, setImageError] = useState(false);

  // Reset image error when URL changes
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset on logoUrl change only
  useEffect(() => {
    setImageError(false);
  }, [logoUrl]);

  // Generate contrasting text color based on background
  const getContrastColor = (hex: string): string => {
    if (!/^#[0-9A-Fa-f]{6}$/.test(hex)) return '#000000';
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
    return luminance > 0.5 ? '#000000' : '#ffffff';
  };

  const textColor = getContrastColor(backgroundColor);
  const mutedTextColor = textColor === '#000000' ? 'rgba(0,0,0,0.6)' : 'rgba(255,255,255,0.7)';

  return (
    <div className="relative">
      <div className="flex items-center gap-2 mb-3">
        <Eye className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium text-muted-foreground">Live Preview</span>
      </div>
      <div
        className="relative rounded-lg border shadow-lg overflow-hidden"
        style={{ backgroundColor, minHeight: '480px' }}
      >
        {/* Custom CSS injection - sanitized to prevent XSS and data exfiltration */}
        {customCss && (
          <style
            // biome-ignore lint/security/noDangerouslySetInnerHtml: CSS is sanitized via sanitizeCSS()
            dangerouslySetInnerHTML={{
              __html: sanitizeCSS(customCss),
            }}
          />
        )}

        <div className="flex flex-col items-center justify-center min-h-[480px] p-6">
          {/* Logo */}
          {logoUrl && !imageError ? (
            <img
              src={logoUrl}
              alt="Logo preview"
              className="max-h-16 max-w-[200px] object-contain mb-6"
              onError={() => setImageError(true)}
            />
          ) : logoUrl && imageError ? (
            <div
              className="mb-6 px-3 py-2 rounded text-xs"
              style={{ backgroundColor: mutedTextColor, color: textColor }}
            >
              Failed to load image
            </div>
          ) : null}

          {/* Title */}
          {title && (
            <h1 className="text-2xl font-bold mb-2 text-center" style={{ color: textColor }}>
              {title}
            </h1>
          )}

          {/* Subtitle */}
          {subtitle && (
            <p className="text-sm mb-4 text-center" style={{ color: mutedTextColor }}>
              {subtitle}
            </p>
          )}

          {/* App Context Message - Shows what resource the user is accessing */}
          <div
            className="flex items-center justify-center gap-2 rounded-lg px-4 py-3 mb-6 w-full max-w-[280px]"
            style={{
              backgroundColor:
                textColor === '#000000' ? 'rgba(0,0,0,0.05)' : 'rgba(255,255,255,0.1)',
              border: `1px solid ${textColor === '#000000' ? 'rgba(0,0,0,0.1)' : 'rgba(255,255,255,0.2)'}`,
            }}
          >
            <Lock className="size-4 flex-shrink-0" style={{ color: mutedTextColor }} />
            <div className="text-center min-w-0">
              <p className="text-xs" style={{ color: mutedTextColor }}>
                You are accessing
              </p>
              <p className="text-sm font-medium truncate" style={{ color: textColor }}>
                internal.company.com
              </p>
            </div>
          </div>

          {/* Mock form - decorative preview only */}
          <div className="w-full max-w-[280px] space-y-4" aria-hidden="true">
            <div className="space-y-2">
              <span className="text-xs font-medium block" style={{ color: mutedTextColor }}>
                Email
              </span>
              <div
                className="flex items-center gap-2 px-3 py-2 rounded-md border"
                style={{
                  borderColor:
                    textColor === '#000000' ? 'rgba(0,0,0,0.2)' : 'rgba(255,255,255,0.3)',
                  backgroundColor:
                    textColor === '#000000' ? 'rgba(0,0,0,0.03)' : 'rgba(255,255,255,0.1)',
                }}
              >
                <Mail className="size-4" style={{ color: mutedTextColor }} />
                <span className="text-sm" style={{ color: mutedTextColor }}>
                  user@example.com
                </span>
              </div>
            </div>

            <div className="space-y-2">
              <span className="text-xs font-medium block" style={{ color: mutedTextColor }}>
                Password
              </span>
              <div
                className="flex items-center gap-2 px-3 py-2 rounded-md border"
                style={{
                  borderColor:
                    textColor === '#000000' ? 'rgba(0,0,0,0.2)' : 'rgba(255,255,255,0.3)',
                  backgroundColor:
                    textColor === '#000000' ? 'rgba(0,0,0,0.03)' : 'rgba(255,255,255,0.1)',
                }}
              >
                <Lock className="size-4" style={{ color: mutedTextColor }} />
                <span className="text-sm" style={{ color: mutedTextColor }}>
                  ************
                </span>
              </div>
            </div>

            {/* Sign in button */}
            <button
              type="button"
              className="w-full py-2 px-4 rounded-md text-sm font-medium transition-opacity hover:opacity-90"
              style={{
                backgroundColor: primaryColor,
                color: getContrastColor(primaryColor),
              }}
            >
              Sign in
            </button>
          </div>

          {/* Footer */}
          {footerText && (
            <p className="text-xs mt-8 text-center" style={{ color: mutedTextColor }}>
              {footerText}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

export function ACLBrandingSettings() {
  const { branding, isLoading } = useACLBranding();
  const { updateBranding, isUpdating } = useUpdateACLBranding();
  const [localValues, setLocalValues] = useState<BrandingFormValues>(DEFAULT_BRANDING);
  const [savedValues, setSavedValues] = useState<BrandingFormValues>(DEFAULT_BRANDING);

  // Initialize local values when branding loads
  useEffect(() => {
    if (branding) {
      const formValues = brandingToForm(branding);
      setLocalValues(formValues);
      setSavedValues(formValues);
    }
  }, [branding]);

  // Track if there are unsaved changes
  const hasChanges = useMemo(() => {
    return (Object.keys(savedValues) as (keyof BrandingFormValues)[]).some(
      (key) => savedValues[key] !== localValues[key],
    );
  }, [localValues, savedValues]);

  // Validate form
  const errors = useMemo(() => {
    const errs: Partial<Record<keyof BrandingFormValues, string>> = {};
    if (localValues.logo_url && !isValidUrl(localValues.logo_url)) {
      errs.logo_url = 'Please enter a valid URL';
    }
    if (!isValidHexColor(localValues.primary_color)) {
      errs.primary_color = 'Please enter a valid hex color';
    }
    if (!isValidHexColor(localValues.background_color)) {
      errs.background_color = 'Please enter a valid hex color';
    }
    return errs;
  }, [localValues]);

  const isValid = Object.keys(errors).length === 0;

  const handleFieldChange = <K extends keyof BrandingFormValues>(
    field: K,
    value: BrandingFormValues[K],
  ) => {
    setLocalValues((prev) => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    if (!isValid) return;
    await updateBranding({
      logo_url: localValues.logo_url || undefined,
      primary_color: localValues.primary_color,
      background_color: localValues.background_color,
      title: localValues.title,
      subtitle: localValues.subtitle || undefined,
      footer_text: localValues.footer_text || undefined,
      custom_css: localValues.custom_css || undefined,
    });
    setSavedValues(localValues);
  };

  const handleReset = () => {
    setLocalValues(savedValues);
  };

  const handleResetToDefaults = () => {
    setLocalValues(DEFAULT_BRANDING);
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>ACL Login Branding</CardTitle>
          <CardDescription>Customize the appearance of the ACL login page.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <div className="space-y-4">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
            <Skeleton className="h-[400px] w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardHeading>
          <CardTitle>ACL Login Branding</CardTitle>
          <CardDescription>
            Customize the appearance of the ACL login page that users see when accessing protected
            resources.
          </CardDescription>
        </CardHeading>
      </CardHeader>
      <CardContent>
        <div className="grid gap-8 lg:grid-cols-2">
          {/* Form Fields */}
          <FieldGroup className="space-y-5">
            {/* Logo URL */}
            <Field data-invalid={!!errors.logo_url}>
              <FieldLabel htmlFor="logo_url">Logo URL</FieldLabel>
              <FieldContent>
                <Input
                  id="logo_url"
                  value={localValues.logo_url}
                  onChange={(e) => handleFieldChange('logo_url', e.target.value)}
                  placeholder="https://example.com/logo.png"
                />
              </FieldContent>
              <FieldDescription>
                URL to your logo image. Recommended size: 200x50px or similar aspect ratio.
              </FieldDescription>
              {errors.logo_url && <FieldError errors={[{ message: errors.logo_url }]} />}
            </Field>

            {/* Primary Color */}
            <ColorInput
              id="primary_color"
              label="Primary Color"
              value={localValues.primary_color}
              onChange={(value) => handleFieldChange('primary_color', value)}
            />

            {/* Background Color */}
            <ColorInput
              id="background_color"
              label="Background Color"
              value={localValues.background_color}
              onChange={(value) => handleFieldChange('background_color', value)}
            />

            {/* Title */}
            <Field>
              <FieldLabel htmlFor="title">Title</FieldLabel>
              <FieldContent>
                <Input
                  id="title"
                  value={localValues.title}
                  onChange={(e) => handleFieldChange('title', e.target.value)}
                  placeholder="Login Required"
                />
              </FieldContent>
              <FieldDescription>Main heading displayed on the login page.</FieldDescription>
            </Field>

            {/* Subtitle */}
            <Field>
              <FieldLabel htmlFor="subtitle">Subtitle</FieldLabel>
              <FieldContent>
                <Input
                  id="subtitle"
                  value={localValues.subtitle}
                  onChange={(e) => handleFieldChange('subtitle', e.target.value)}
                  placeholder="Sign in to continue"
                />
              </FieldContent>
              <FieldDescription>Secondary text shown below the title.</FieldDescription>
            </Field>

            {/* Footer Text */}
            <Field>
              <FieldLabel htmlFor="footer_text">Footer Text</FieldLabel>
              <FieldContent>
                <Input
                  id="footer_text"
                  value={localValues.footer_text}
                  onChange={(e) => handleFieldChange('footer_text', e.target.value)}
                  placeholder="© 2025 Your Company"
                />
              </FieldContent>
              <FieldDescription>Text displayed at the bottom of the login page.</FieldDescription>
            </Field>

            {/* Custom CSS (Advanced) */}
            <Field>
              <FieldLabel htmlFor="custom_css">Custom CSS (Advanced)</FieldLabel>
              <FieldContent>
                <Textarea
                  id="custom_css"
                  value={localValues.custom_css}
                  onChange={(e) => handleFieldChange('custom_css', e.target.value)}
                  placeholder={`.login-container {\n  /* Your custom styles */\n}`}
                  className="font-mono min-h-[120px]"
                  rows={5}
                />
              </FieldContent>
              <FieldDescription>
                Add custom CSS to further customize the login page appearance.
              </FieldDescription>
            </Field>
          </FieldGroup>

          {/* Live Preview */}
          <div className="lg:sticky lg:top-6">
            <LoginPreview
              logoUrl={localValues.logo_url}
              primaryColor={localValues.primary_color}
              backgroundColor={localValues.background_color}
              title={localValues.title}
              subtitle={localValues.subtitle}
              footerText={localValues.footer_text}
              customCss={localValues.custom_css}
            />
          </div>
        </div>
      </CardContent>
      <CardFooter className="flex justify-between border-t">
        <Button variant="ghost" onClick={handleResetToDefaults} disabled={isUpdating}>
          <RotateCcw className="size-4" />
          Reset to Defaults
        </Button>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleReset} disabled={isUpdating || !hasChanges}>
            Discard Changes
          </Button>
          <Button onClick={handleSave} disabled={isUpdating || !hasChanges || !isValid}>
            {isUpdating ? (
              <>
                <Spinner variant="circle" />
                Saving...
              </>
            ) : (
              'Save Changes'
            )}
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}

// Utility functions
function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

function isValidHexColor(color: string): boolean {
  return /^#[0-9A-Fa-f]{6}$/.test(color);
}
