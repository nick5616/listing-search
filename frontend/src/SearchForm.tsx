import type { FormEvent } from "react";
import type { SearchForm as SearchFormValues } from "./types";

const PAGE_SIZES = ["5", "10", "20"];

interface Props {
  form: SearchFormValues;
  onChange: (name: keyof SearchFormValues, value: string) => void;
  onSubmit: (e: FormEvent) => void;
  onReset: () => void;
  errors: Record<string, string>; // server messages keyed by field name
  cities: string[];
}

// Inputs only. There are no client-side rules: the server validates and its
// messages come back in `errors`, shown under the matching field. Searches run
// as the user types; pressing Enter submits so the search runs without waiting.
export function SearchForm({ form, onChange, onSubmit, onReset, errors, cities }: Props) {
  const field = (name: keyof SearchFormValues) => ({
    name,
    value: form[name],
    onChange: (v: string) => onChange(name, v),
    error: errors[name],
  });

  return (
    <form className="panel search" onSubmit={onSubmit} noValidate>
      <div className="search-grid">
        <Field label="Keyword" {...field("keyword")} placeholder="e.g. yard, metro" wide />
        <Field label="Address" {...field("address")} placeholder="e.g. main st" wide />
        <Field label="City" {...field("city")} list="city-options" placeholder="Any city" />
        <datalist id="city-options">
          {cities.map((c) => <option key={c} value={c} />)}
        </datalist>
        <Field label="Status" {...field("status")} select={[["", "Any"], ["active", "Active"], ["pending", "Pending"], ["sold", "Sold"]]} />

        <Field label="Min price" {...field("minPrice")} inputMode="numeric" placeholder="No min" prefix="$" />
        <Field label="Max price" {...field("maxPrice")} inputMode="numeric" placeholder="No max" prefix="$" />
        <Field label="Min bedrooms" {...field("minBedrooms")} inputMode="numeric" placeholder="Any" />
        <Field label="Target budget" {...field("targetBudget")} inputMode="numeric" placeholder="Optional" prefix="$"
          hint="Ranks by how close each price is to this" />
      </div>

      <div className="search-actions">
        <Field label="Per page" {...field("pageSize")} select={PAGE_SIZES.map((n) => [n, n])} compact />
        <div className="buttons">
          <button type="button" className="btn btn-secondary" onClick={onReset}>Clear</button>
        </div>
      </div>
    </form>
  );
}

function Field(props: {
  label: string;
  name: string;
  value: string;
  onChange: (v: string) => void;
  error?: string;
  hint?: string;
  placeholder?: string;
  list?: string;
  inputMode?: "numeric" | "text";
  prefix?: string;
  select?: string[][]; // [value, label] pairs; renders a <select> instead of an <input>
  wide?: boolean;
  compact?: boolean;
}) {
  const errorId = `${props.name}-error`;
  const hintId = `${props.name}-hint`;
  const describedBy = [props.error && errorId, props.hint && hintId].filter(Boolean).join(" ") || undefined;
  const shared = {
    name: props.name,
    value: props.value,
    "aria-invalid": !!props.error,
    "aria-describedby": describedBy,
  };

  return (
    <label className={`field${props.wide ? " wide" : ""}${props.compact ? " compact" : ""}`}>
      <span className="field-label">{props.label}</span>
      <span className="control">
        {props.prefix && <span className="prefix" aria-hidden="true">{props.prefix}</span>}
        {props.select ? (
          <select {...shared} onChange={(e) => props.onChange(e.target.value)}>
            {props.select.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        ) : (
          <input {...shared} onChange={(e) => props.onChange(e.target.value)} placeholder={props.placeholder}
            list={props.list} inputMode={props.inputMode} autoComplete="off" />
        )}
      </span>
      {props.error
        ? <small id={errorId} className="field-error">{props.error}</small>
        : props.hint && <small id={hintId} className="field-hint">{props.hint}</small>}
    </label>
  );
}
