import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  FieldGroup,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { api } from '@/lib/api';

const passwordSchema = z
  .object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(8, 'Password must be at least 8 characters'),
    confirm_password: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords don't match",
    path: ['confirm_password'],
  });
type PasswordValues = z.infer<typeof passwordSchema>;

interface ChangePasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * When true, the dialog cannot be dismissed (no Cancel button, overlay/Esc
   * clicks are ignored). Used when the server requires the user to change their
   * password before proceeding.
   */
  forced?: boolean;
}

export function ChangePasswordDialog({ open, onOpenChange, forced }: ChangePasswordDialogProps) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const form = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    mode: 'onTouched',
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    },
  });

  const mutation = useMutation({
    mutationFn: async (data: { current_password: string; new_password: string }) => {
      return await api.post('auth/change-password', { json: data }).json();
    },
    onSuccess: () => {
      setStatus({ type: 'success', message: 'Password changed successfully!' });
      form.reset();
      // Invalidate /auth/me so the must_change_password gate re-evaluates
      void queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
      if (!forced) {
        setTimeout(() => {
          onOpenChange(false);
          setStatus(null);
        }, 1500);
      }
    },
    onError: (error: Error) => {
      setStatus({ type: 'error', message: error.message || 'Failed to change password' });
    },
  });

  const onSubmit = (value: PasswordValues) => {
    setStatus(null);
    mutation.mutate({
      current_password: value.current_password,
      new_password: value.new_password,
    });
  };

  const handleOpenChange = (isOpen: boolean) => {
    // In forced mode, prevent any close attempt
    if (forced && !isOpen) return;
    if (!isOpen) {
      setStatus(null);
      form.reset();
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} disablePointerDismissal={forced}>
      <DialogContent showCloseButton={!forced}>
        <DialogHeader>
          <DialogTitle>Change Password</DialogTitle>
          <DialogDescription>
            {forced
              ? 'You must change your password before you can use the app.'
              : 'Enter your current password and choose a new one.'}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <FieldGroup>
                {status && (
                  <Alert variant={status.type === 'error' ? 'destructive' : 'success'}>
                    {status.type === 'success' ? (
                      <CheckCircle2 className="size-4" />
                    ) : (
                      <XCircle className="size-4" />
                    )}
                    <AlertDescription>{status.message}</AlertDescription>
                  </Alert>
                )}

                <FormField
                  control={form.control}
                  name="current_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Current Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="current-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="new_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>New Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="confirm_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Confirm New Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </FieldGroup>
            </div>
            <DialogFooter>
              {!forced && (
                <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                  Cancel
                </Button>
              )}
              <Button type="submit" disabled={mutation.isPending || status?.type === 'success'}>
                {mutation.isPending ? 'Changing...' : 'Change Password'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
