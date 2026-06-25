import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Skeleton,
  Spinner,
  Textarea,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Eye, Lock, RotateCcw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { useACLBranding, useUpdateACLBranding } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';
import { sanitizeCSS } from '@/lib/css-sanitizer';
import type { ACLBranding } from '@/types/acl';

const brandingSchema = z.object({
  logo_url: z
    .string()
    .optional()
    .refine((val) => !val || isValidUrl(val), { message: 'Please enter a valid URL' }),
  primary_color: z
    .string()
    .refine((val) => isValidHexColor(val), { message: 'Please enter a valid hex color' }),
  background_color: z
    .string()
    .optional()
    .refine((val) => !val || isValidHexColor(val), { message: 'Please enter a valid hex color' }),
  title: z.string().optional(),
  subtitle: z.string().optional(),
  footer_text: z.string().optional(),
  custom_css: z.string().optional(),
});

type BrandingFormValues = z.infer<typeof brandingSchema>;

const DEFAULT_BRANDING: BrandingFormValues = {
  logo_url: '',
  primary_color: '#6E72F0',
  background_color: '',
  title: 'Waygates',
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
  errorMessage,
}: {
  value: string;
  onChange: (value: string) => void;
  id: string;
  label: string;
  errorMessage?: string;
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

  const showError = (!isValidHex && inputValue.length > 0) || !!errorMessage;

  return (
    <div data-invalid={showError || undefined} className="space-y-2">
      <label htmlFor={id} className="text-sm font-medium leading-none">
        {label}
      </label>
      <div className="flex gap-2">
        <Input
          id={id}
          value={inputValue}
          onChange={(e) => handleInputChange(e.target.value)}
          placeholder="#000000"
          className="flex-1 font-mono"
          aria-invalid={showError}
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
      {!isValidHex && inputValue.length > 0 && (
        <p className="text-sm text-destructive">Please enter a valid hex color (e.g., #6E72F0)</p>
      )}
    </div>
  );
}

// Contrasting text color for custom backgrounds
function getContrastColor(hex: string): string {
  if (!/^#[0-9A-Fa-f]{6}$/.test(hex)) return '#000000';
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  return luminance > 0.5 ? '#000000' : '#ffffff';
}

// Login page preview — mirrors the actual ACL login page structure
// Uses the same Card/CardContent layout and Tailwind classes as the real page
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

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset on logoUrl change only
  useEffect(() => {
    setImageError(false);
  }, [logoUrl]);

  const hasCustomBg = backgroundColor && /^#[0-9A-Fa-f]{6}$/.test(backgroundColor);
  const validPrimary = /^#[0-9A-Fa-f]{6}$/.test(primaryColor);

  return (
    <div className="relative">
      <div className="flex items-center gap-2 mb-3">
        <Eye className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium text-muted-foreground">Live Preview</span>
      </div>

      {/* Outer page container — mirrors acl-login.tsx layout */}
      <div
        className="relative rounded border shadow-lg overflow-hidden bg-background"
        style={{ ...(hasCustomBg ? { backgroundColor } : {}), minHeight: '480px' }}
      >
        {customCss && (
          <style
            // biome-ignore lint/security/noDangerouslySetInnerHtml: CSS is sanitized via sanitizeCSS()
            dangerouslySetInnerHTML={{ __html: sanitizeCSS(customCss) }}
          />
        )}

        {/* Centered card — same as the real page */}
        <div className="flex items-center justify-center min-h-[480px] px-4 py-8">
          <Card className="w-full max-w-xs shadow-lg">
            <CardContent className="pt-6">
              <div className="space-y-6">
                {/* Header — matches ACLLoginContent */}
                <div className="text-center space-y-2">
                  {logoUrl && !imageError ? (
                    <img
                      src={logoUrl}
                      alt="Logo preview"
                      className="h-12 w-auto mx-auto object-contain"
                      onError={() => setImageError(true)}
                    />
                  ) : logoUrl && imageError ? (
                    <p className="text-xs text-muted-foreground">Failed to load image</p>
                  ) : (
                    <div className="h-12 w-12 mx-auto rounded bg-primary/10 flex items-center justify-center">
                      <Lock className="size-6 text-primary" />
                    </div>
                  )}
                  {title && <h1 className="text-xl font-semibold tracking-tight">{title}</h1>}
                  {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
                </div>

                {/* Host badge — matches HostBadge component */}
                <div className="flex items-center justify-center gap-2 rounded border bg-muted/50 px-4 py-3">
                  <Lock className="size-4 text-muted-foreground" />
                  <div className="text-center">
                    <p className="text-xs text-muted-foreground">Accessing</p>
                    <p className="text-sm font-medium">internal.company.com</p>
                  </div>
                </div>

                {/* Mock form — matches ACLLoginForm structure */}
                <div className="space-y-4" aria-hidden="true">
                  <div className="space-y-2">
                    <span className="block text-sm font-medium">Username or Email</span>
                    <div className="flex h-9 w-full items-center rounded border border-input bg-transparent px-3 text-sm text-muted-foreground">
                      user@example.com
                    </div>
                  </div>

                  <div className="space-y-2">
                    <span className="block text-sm font-medium">Password</span>
                    <div className="flex h-9 w-full items-center rounded border border-input bg-transparent px-3 text-sm text-muted-foreground">
                      ************
                    </div>
                  </div>

                  <button
                    type="button"
                    className="w-full h-9 rounded text-sm font-medium transition-opacity hover:opacity-90"
                    style={{
                      backgroundColor: validPrimary ? primaryColor : 'var(--primary)',
                      color: validPrimary
                        ? getContrastColor(primaryColor)
                        : 'var(--primary-foreground)',
                    }}
                  >
                    Sign in
                  </button>
                </div>

                {/* Footer */}
                {footerText && (
                  <p className="text-center text-xs text-muted-foreground">{footerText}</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export function ACLBrandingSettings() {
  const { branding, isLoading } = useACLBranding();
  const { updateBranding, isUpdating } = useUpdateACLBranding();
  const { canUpdateAccess } = usePermissions();

  const form = useForm<BrandingFormValues>({
    resolver: zodResolver(brandingSchema),
    defaultValues: DEFAULT_BRANDING,
  });

  const { isDirty, isValid } = form.formState;

  // Seed form from loaded branding
  useEffect(() => {
    if (branding) {
      form.reset(brandingToForm(branding));
    }
  }, [branding, form]);

  // Drive the live preview from RHF watch
  const watched = form.watch();

  const onSubmit = async (values: BrandingFormValues) => {
    await updateBranding({
      logo_url: values.logo_url || undefined,
      primary_color: values.primary_color,
      background_color: values.background_color,
      title: values.title,
      subtitle: values.subtitle || undefined,
      footer_text: values.footer_text || undefined,
      custom_css: values.custom_css || undefined,
    });
    // Clear dirty state — mirrors original setSavedValues(localValues)
    form.reset(values);
  };

  const handleResetToDefaults = () => {
    form.reset(DEFAULT_BRANDING, { keepDefaultValues: true });
  };

  const handleDiscard = () => {
    form.reset();
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Login Branding</CardTitle>
          <CardDescription>Customize the login page appearance.</CardDescription>
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
        <CardTitle>Login Branding</CardTitle>
        <CardDescription>
          Customize the login page that users see when accessing protected resources.
        </CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent>
            <div className="grid gap-8 lg:grid-cols-2">
              {/* Form Fields */}
              <div className="space-y-5">
                {/* Logo URL */}
                <FormField
                  control={form.control}
                  name="logo_url"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Logo URL</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="https://example.com/logo.png"
                          {...field}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FormDescription>
                        URL to your logo image. Recommended size: 200x50px or similar aspect ratio.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* Primary Color */}
                <FormField
                  control={form.control}
                  name="primary_color"
                  render={({ field, fieldState }) => (
                    <FormItem>
                      <FormControl>
                        <ColorInput
                          id="primary_color"
                          label="Primary Color"
                          value={field.value ?? ''}
                          onChange={field.onChange}
                          errorMessage={fieldState.error?.message}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                {/* Background Color */}
                <FormField
                  control={form.control}
                  name="background_color"
                  render={({ field, fieldState }) => (
                    <FormItem>
                      <FormControl>
                        <ColorInput
                          id="background_color"
                          label="Background Color"
                          value={field.value ?? ''}
                          onChange={field.onChange}
                          errorMessage={fieldState.error?.message}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                {/* Title */}
                <FormField
                  control={form.control}
                  name="title"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Title</FormLabel>
                      <FormControl>
                        <Input placeholder="Login Required" {...field} value={field.value ?? ''} />
                      </FormControl>
                      <FormDescription>Main heading displayed on the login page.</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* Subtitle */}
                <FormField
                  control={form.control}
                  name="subtitle"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Subtitle</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="Sign in to continue"
                          {...field}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FormDescription>Secondary text shown below the title.</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* Footer Text */}
                <FormField
                  control={form.control}
                  name="footer_text"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Footer Text</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="© 2025 Your Company"
                          {...field}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FormDescription>
                        Text displayed at the bottom of the login page.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* Custom CSS (Advanced) */}
                <FormField
                  control={form.control}
                  name="custom_css"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Custom CSS (Advanced)</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder={`.login-container {\n  /* Your custom styles */\n}`}
                          className="font-mono min-h-[120px]"
                          rows={5}
                          {...field}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FormDescription>
                        Add custom CSS to further customize the login page appearance.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {/* Live Preview — driven by form.watch() */}
              <div className="lg:sticky lg:top-6">
                <LoginPreview
                  logoUrl={watched.logo_url ?? ''}
                  primaryColor={watched.primary_color ?? ''}
                  backgroundColor={watched.background_color ?? ''}
                  title={watched.title ?? ''}
                  subtitle={watched.subtitle ?? ''}
                  footerText={watched.footer_text ?? ''}
                  customCss={watched.custom_css ?? ''}
                />
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between border-t">
            <Button
              type="button"
              variant="ghost"
              onClick={handleResetToDefaults}
              disabled={!canUpdateAccess || isUpdating}
            >
              <RotateCcw className="size-4" />
              Reset to Defaults
            </Button>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleDiscard}
                disabled={isUpdating || !isDirty}
              >
                Discard Changes
              </Button>
              <Button
                type="submit"
                disabled={!canUpdateAccess || isUpdating || !isDirty || !isValid}
              >
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
        </form>
      </Form>
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
