import type { ZodSchema } from 'zod';

/**
 * Creates a TanStack Form validator from a Zod schema.
 * This is needed because @tanstack/zod-form-adapter doesn't support Zod 4.x yet.
 */
export function zodValidator<T>(schema: ZodSchema<T>) {
  return {
    validate: ({ value }: { value: T }) => {
      const result = schema.safeParse(value);
      if (result.success) {
        return undefined;
      }
      // Zod 4 uses 'issues' instead of 'errors'
      return result.error.issues.map((issue) => issue.message).join(', ');
    },
    validateAsync: async ({ value }: { value: T }) => {
      const result = await schema.safeParseAsync(value);
      if (result.success) {
        return undefined;
      }
      return result.error.issues.map((issue) => issue.message).join(', ');
    },
  };
}
