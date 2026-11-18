import { useEffect, useId, useRef, useState, type ComponentProps } from 'react';

import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from '@/components/ui/combobox';
import { FieldError } from '@/components/ui/field';

interface TagsInputProps extends Pick<
  ComponentProps<'input'>,
  'aria-label' | 'aria-describedby' | 'aria-invalid' | 'onBlur' | 'disabled'
> {
  value: string[];
  onValueChange: (value: string[]) => void;
  placeholder?: string;
  validation?: { pattern: RegExp };
  delimiters?: string[];
  suggestions?: string[];
  className?: string;
  id?: string;
}
export function TagsInput({
  value,
  onValueChange,
  placeholder,
  validation,
  delimiters = ['Enter', ','],
  suggestions = [],
  className,
  id,
  onBlur,
  ...inputProps
}: TagsInputProps) {
  const anchor = useComboboxAnchor();
  const generatedId = useId();
  const [inputValue, setInputValue] = useState('');
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    inputRef.current?.setCustomValidity(error);
  }, [error]);
  const hasHighlightRef = useRef(false);
  const errorId = `${id || generatedId}-error`;
  function addTags(raw: string) {
    const tokens = raw
      .split(/[,;\n]/)
      .map((v) => v.trim())
      .filter(Boolean);
    if (!tokens.length) return;
    if (validation?.pattern && tokens.some((v) => !new RegExp(validation.pattern).test(v))) {
      setError('Enter a valid value.');
      return;
    }
    const next = [...new Set([...value, ...tokens])];
    if (next.length !== value.length) onValueChange(next);
    setInputValue('');
    setError('');
  }
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Combobox
        items={suggestions}
        multiple
        value={value}
        onValueChange={(v) => onValueChange(v ?? [])}
        inputValue={inputValue}
        onInputValueChange={(v) => {
          setInputValue(v);
          setError('');
        }}
        onItemHighlighted={(item) => {
          hasHighlightRef.current = item != null;
        }}
        {...(!suggestions.length ? { open: false } : {})}
      >
        <ComboboxChips ref={anchor} className={className}>
          <ComboboxValue>
            {(tags: string[]) => (
              <>
                {tags.map((tag) => (
                  <ComboboxChip key={tag} aria-label={tag}>
                    {tag}
                  </ComboboxChip>
                ))}
                <ComboboxChipsInput
                  ref={inputRef}
                  {...inputProps}
                  id={id || generatedId}
                  placeholder={tags.length ? '' : placeholder}
                  aria-invalid={!!error || inputProps['aria-invalid']}
                  aria-describedby={error ? errorId : inputProps['aria-describedby']}
                  onBlur={(e) => {
                    if (inputValue.trim()) addTags(inputValue);
                    onBlur?.(e);
                  }}
                  onPaste={(e) => {
                    const text = e.clipboardData.getData('text');
                    if (/[,;\n]/.test(text)) {
                      e.preventDefault();
                      addTags(text);
                    }
                  }}
                  onKeyDown={(e) => {
                    if (
                      !e.nativeEvent.isComposing &&
                      delimiters.includes(e.key) &&
                      !hasHighlightRef.current &&
                      inputValue.trim()
                    ) {
                      e.preventDefault();
                      addTags(inputValue);
                    }
                  }}
                />
              </>
            )}
          </ComboboxValue>
        </ComboboxChips>
        {suggestions.length > 0 && (
          <ComboboxContent anchor={anchor}>
            <ComboboxList>
              {(item: string) => (
                <ComboboxItem key={item} value={item}>
                  {item}
                </ComboboxItem>
              )}
            </ComboboxList>
          </ComboboxContent>
        )}
      </Combobox>
      {error && <FieldError id={errorId}>{error}</FieldError>}
    </div>
  );
}
