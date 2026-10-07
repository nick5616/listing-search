import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ApiError, fetchCities, searchListings } from "./api";
import { Results } from "./Results";
import { SearchForm } from "./SearchForm";
import type { SearchForm as SearchFormValues, SearchParams, SearchResponse } from "./types";

const emptyForm: SearchFormValues = {
  keyword: "",
  address: "",
  city: "",
  minPrice: "",
  maxPrice: "",
  minBedrooms: "",
  targetBudget: "",
  status: "",
  pageSize: "5",
};

// How long typing has to pause before the search runs.
const DEBOUNCE_MS = 300;

// Compares only the form's fields, so `params.page` doesn't count as a change.
function sameFields(form: SearchFormValues, params: SearchParams): boolean {
  return (Object.keys(form) as (keyof SearchFormValues)[]).every((k) => form[k] === params[k]);
}

export default function App() {
  // `form` is what the user is typing; `params` is the search actually sent.
  // `params` follows `form` after a short pause in typing, so the results update
  // live without a request on every keystroke.
  const [form, setForm] = useState<SearchFormValues>(emptyForm);
  const [params, setParams] = useState<SearchParams>({ ...emptyForm, page: 1 });

  const [data, setData] = useState<SearchResponse | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(true);
  const [cities, setCities] = useState<string[]>([]);

  useEffect(() => {
    const controller = new AbortController();
    fetchCities(controller.signal).then(setCities).catch(() => {
      /* suggestions are optional; search still works without them */
    });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => runSearch(form), DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [form]);

  useEffect(() => {
    // Abort the previous request if params change before it finishes,
    // so a slow old response can never overwrite a newer one.
    const controller = new AbortController();
    setLoading(true);
    searchListings(params, controller.signal)
      .then((res) => {
        setData(res);
        setError(null);
      })
      .catch((err: Error) => {
        if (err.name === "AbortError") return;
        setError(err instanceof ApiError ? err : new ApiError(err.message, 0));
        setData(null);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [params]);

  const fieldErrors = useMemo(
    () => Object.fromEntries((error?.details ?? []).map((d) => [d.field, d.message])),
    [error],
  );

  // New filters always start at page 1. Unchanged filters keep the current
  // params (and page), so a debounce that fires after Enter or Clear is a no-op.
  function runSearch(values: SearchFormValues) {
    setParams((p) => (sameFields(values, p) ? p : { ...values, page: 1 }));
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    runSearch(form);
  }

  function onReset() {
    setForm(emptyForm);
    setParams({ ...emptyForm, page: 1 });
  }

  const goToPage = (page: number) => setParams((p) => ({ ...p, page }));

  return (
    <>
      <header className="topbar">
        <div className="container">
          <span className="brand">Listing Search</span>
        </div>
      </header>

      <main className="container">
        <SearchForm
          form={form}
          onChange={(name, value) => setForm((f) => ({ ...f, [name]: value }))}
          onSubmit={onSubmit}
          onReset={onReset}
          errors={fieldErrors}
          cities={cities}
        />
        <Results data={data} error={error} loading={loading} onPage={goToPage} onReset={onReset} />
      </main>
    </>
  );
}
