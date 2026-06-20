import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from '@e412/rnui-react';
import { useRef, useState } from 'react';

interface TagsInputProps {
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
  delimiters = ['Enter'],
  suggestions = [],
  className,
  id,
}: TagsInputProps) {
  const anchor = useComboboxAnchor();
  const [inputValue, setInputValue] = useState('');
  const hasHighlightRef = useRef(false);

  function addTag(raw: string) {
    const next = raw.trim();
    if (next === '') return;
    if (value.includes(next)) return;
    if (validation?.pattern && !validation.pattern.test(next)) return;
    onValueChange([...value, next]);
    setInputValue('');
  }

  return (
    <Combobox
      items={suggestions}
      multiple
      value={value}
      onValueChange={(newValue) => {
        // newValue is string[] | null (multiple mode)
        onValueChange((newValue as string[]) ?? []);
      }}
      inputValue={inputValue}
      onInputValueChange={(v) => setInputValue(v)}
      onItemHighlighted={(item) => {
        hasHighlightRef.current = item != null;
      }}
    >
      <ComboboxChips ref={anchor} className={className} id={id}>
        <ComboboxValue>
          {(tags: string[]) => (
            <>
              {tags.map((tag) => (
                <ComboboxChip key={tag} aria-label={tag}>
                  {tag}
                </ComboboxChip>
              ))}
              <ComboboxChipsInput
                placeholder={tags.length > 0 ? '' : placeholder}
                onKeyDown={(event) => {
                  const key = event.key;
                  const isDelimiter = delimiters.includes(key);
                  if (isDelimiter && !hasHighlightRef.current && inputValue.trim() !== '') {
                    event.preventDefault();
                    addTag(inputValue);
                  }
                }}
              />
            </>
          )}
        </ComboboxValue>
      </ComboboxChips>
      <ComboboxContent anchor={anchor}>
        <ComboboxEmpty>
          {inputValue.trim() === ''
            ? 'No suggestions.'
            : `Press Enter to add "${inputValue.trim()}"`}
        </ComboboxEmpty>
        <ComboboxList>
          {(item: string) => (
            <ComboboxItem key={item} value={item}>
              {item}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
