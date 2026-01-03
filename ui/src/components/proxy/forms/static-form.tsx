import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import {
  Button,
  Input,
  Switch,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  CardToolbar,
  Field,
  FieldContent,
  FieldLabel,
  FieldError,
  FieldDescription,
} from '@e412/titanium';
import { Plus, Trash2 } from 'lucide-react';
import type { CreateStaticRequest, Proxy } from '@/types/proxy';

const staticSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z.string().min(1, 'Hostname is required').max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  root_path: z.string().min(1, 'Root path is required'),
  index_file: z.string().min(1, 'Index file is required'),
  browse: z.boolean(),
  template_rendering: z.boolean(),
});

type StaticFormValues = z.infer<typeof staticSchema>;

interface StaticFormProps {
  initialData?: Proxy | null;
  onSubmit: (data: CreateStaticRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

export function StaticForm({ initialData, onSubmit, loading, onCancel }: StaticFormProps) {
  const [tryFiles, setTryFiles] = useState<string[]>([]);

  const form = useForm({
    defaultValues: {
      name: '',
      hostname: '',
      description: '',
      root_path: '/var/www/html',
      index_file: 'index.html',
      browse: false,
      template_rendering: false,
    } as StaticFormValues,
    validators: {
      onChange: staticSchema,
    },
    onSubmit: async ({ value }) => {
      const data: CreateStaticRequest = {
        type: 'static',
        name: value.name,
        hostname: value.hostname,
        description: value.description || undefined,
        static: {
          root_path: value.root_path,
          index_file: value.index_file,
          browse: value.browse,
          template_rendering: value.template_rendering,
          try_files: tryFiles.length ? tryFiles.filter(Boolean) : undefined,
        },
      };

      onSubmit(data);
    },
  });

  useEffect(() => {
    if (initialData) {
      form.setFieldValue('name', initialData.name || '');
      form.setFieldValue('hostname', initialData.hostname || '');
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('root_path', initialData.static?.root_path || '/var/www/html');
      form.setFieldValue('index_file', initialData.static?.index_file || 'index.html');
      form.setFieldValue('browse', initialData.static?.browse ?? false);
      form.setFieldValue('template_rendering', initialData.static?.template_rendering ?? false);
      setTryFiles(initialData.static?.try_files || []);
    }
  }, [initialData]);

  const addTryFile = () => {
    setTryFiles([...tryFiles, '']);
  };

  const removeTryFile = (index: number) => {
    setTryFiles(tryFiles.filter((_, i) => i !== index));
  };

  const updateTryFile = (index: number, value: string) => {
    const newTryFiles = [...tryFiles];
    newTryFiles[index] = value;
    setTryFiles(newTryFiles);
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
      className="space-y-6"
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <form.Field name="name">
          {(field) => {
            const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
            return (
              <Field data-invalid={hasError}>
                <FieldLabel htmlFor={field.name}>Name</FieldLabel>
                <Input
                  id={field.name}
                  placeholder="Documentation Site"
                  value={field.state.value}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  aria-invalid={hasError}
                />
                {hasError && <FieldError errors={field.state.meta.errors} />}
              </Field>
            );
          }}
        </form.Field>

        <form.Field name="hostname">
          {(field) => {
            const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
            return (
              <Field data-invalid={hasError}>
                <FieldLabel htmlFor={field.name}>Hostname</FieldLabel>
                <Input
                  id={field.name}
                  placeholder="docs.example.com"
                  value={field.state.value}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  aria-invalid={hasError}
                />
                {hasError && <FieldError errors={field.state.meta.errors} />}
              </Field>
            );
          }}
        </form.Field>
      </div>

      <form.Field name="description">
        {(field) => {
          const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
          return (
            <Field data-invalid={hasError}>
              <FieldLabel htmlFor={field.name}>Description (optional)</FieldLabel>
              <Input
                id={field.name}
                placeholder="Static documentation files"
                value={field.state.value}
                onChange={(e) => field.handleChange(e.target.value)}
                onBlur={field.handleBlur}
                aria-invalid={hasError}
              />
              {hasError && <FieldError errors={field.state.meta.errors} />}
            </Field>
          );
        }}
      </form.Field>

      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>File Server Configuration</CardTitle>
            <CardDescription>Configure how static files are served to visitors</CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          <form.Field name="root_path">
            {(field) => {
              const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
              return (
                <Field data-invalid={hasError}>
                  <FieldLabel htmlFor={field.name}>Root Path</FieldLabel>
                  <Input
                    id={field.name}
                    placeholder="/var/www/html"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    onBlur={field.handleBlur}
                    aria-invalid={hasError}
                  />
                  <FieldDescription>
                    The directory path to serve files from
                  </FieldDescription>
                  {hasError && <FieldError errors={field.state.meta.errors} />}
                </Field>
              );
            }}
          </form.Field>

          <form.Field name="index_file">
            {(field) => {
              const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
              return (
                <Field data-invalid={hasError}>
                  <FieldLabel htmlFor={field.name}>Index File</FieldLabel>
                  <Input
                    id={field.name}
                    placeholder="index.html"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    onBlur={field.handleBlur}
                    aria-invalid={hasError}
                  />
                  <FieldDescription>
                    Default file to serve for directory requests
                  </FieldDescription>
                  {hasError && <FieldError errors={field.state.meta.errors} />}
                </Field>
              );
            }}
          </form.Field>

          <form.Field name="browse">
            {(field) => (
              <Field orientation="horizontal">
                <FieldContent>
                  <FieldLabel>Directory Browsing</FieldLabel>
                  <FieldDescription>
                    Allow visitors to browse directory contents
                  </FieldDescription>
                </FieldContent>
                <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
              </Field>
            )}
          </form.Field>

          <form.Field name="template_rendering">
            {(field) => (
              <Field orientation="horizontal">
                <FieldContent>
                  <FieldLabel>Template Rendering</FieldLabel>
                  <FieldDescription>
                    Enable Caddy template rendering for HTML files
                  </FieldDescription>
                </FieldContent>
                <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
              </Field>
            )}
          </form.Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Try Files (Optional)</CardTitle>
            <CardDescription>Specify fallback files when the requested path is not found</CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button type="button" variant="outline" size="sm" onClick={addTryFile}>
              <Plus className="mr-1 size-4" />
              Add File
            </Button>
          </CardToolbar>
        </CardHeader>
        <CardContent className="space-y-3">
          {tryFiles.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No try_files configured. Files will be served directly if they exist.
            </p>
          ) : (
            <>
              {tryFiles.map((file, index) => (
                <div key={index} className="flex items-center gap-2">
                  <Input
                    placeholder="{path} or /index.html"
                    value={file}
                    onChange={(e) => updateTryFile(index, e.target.value)}
                    className="flex-1"
                  />
                  <Button type="button" variant="ghost" size="icon" onClick={() => removeTryFile(index)}>
                    <Trash2 className="size-4 text-destructive" />
                  </Button>
                </div>
              ))}
              <p className="text-sm text-muted-foreground">
                Caddy will try each file in order. Use {'{path}'} for the original request path.
              </p>
            </>
          )}
        </CardContent>
      </Card>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : 'Save'}
        </Button>
      </div>
    </form>
  );
}
