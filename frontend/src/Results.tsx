import type { ApiError } from "./api";
import type { Listing, SearchResponse } from "./types";

const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });

interface Props {
  data: SearchResponse | null;
  error: ApiError | null;
  loading: boolean;
  onPage: (page: number) => void;
  onReset: () => void;
}

export function Results({ data, error, loading, onPage, onReset }: Props) {
  return (
    <section className="results" aria-live="polite" aria-busy={loading}>
      {error && <ErrorBanner error={error} />}

      {data?.warnings.map((w) => (
        <p key={w.field + w.message} className="banner warning" role="status">{w.message}</p>
      ))}

      {loading && !data && <p className="muted">Searching...</p>}

      {data && data.total === 0 && (
        <div className="panel empty">
          <h2>No listings match these filters</h2>
          <p className="muted">Try widening the price range or removing a filter.</p>
          <button type="button" className="btn btn-secondary" onClick={onReset}>Clear filters</button>
        </div>
      )}

      {data && data.total > 0 && (
        <>
          <Summary data={data} />
          <ol className={`listings${loading ? " dimmed" : ""}`}>
            {data.results.map((l) => <ListingRow key={l.key} listing={l} />)}
          </ol>
          <Pagination data={data} disabled={loading} onPage={onPage} />
        </>
      )}
    </section>
  );
}

function Summary({ data }: { data: SearchResponse }) {
  const { page, pageSize, total } = data;
  const first = (page - 1) * pageSize + 1;
  const last = Math.min(page * pageSize, total);
  return (
    <div className="summary">
      <h2>{total} {total === 1 ? "listing" : "listings"}</h2>
      <span className="muted">Showing {first} to {last}, highest relevance first</span>
    </div>
  );
}

function ListingRow({ listing: l }: { listing: Listing }) {
  const { budgetFit, recency } = l.scoreBreakdown;
  return (
    <li className="panel listing">
      <div className="score" title="Relevance score, 0 to 100">
        <span className="score-value">{l.score.toFixed(1)}</span>
        <span className="score-label">relevance</span>
        <span className="score-parts">
          {budgetFit !== undefined && <span>budget {budgetFit.toFixed(2)}</span>}
          <span>recency {recency.toFixed(2)}</span>
        </span>
      </div>

      <div className="listing-main">
        <h3>{l.address}</h3>
        <p className="muted">{l.city}, {l.state} {l.zip}</p>
        <p className="description">{l.description}</p>
        <p className="meta">
          <span className={`status ${l.status}`}>{l.status}</span>
          <span>Listed {l.listedDate}</span>
          <span>{l.key}</span>
        </p>
      </div>

      <div className="listing-facts">
        <span className="price">{usd.format(l.price)}</span>
        <span>{l.bedrooms} bd · {l.bathrooms} ba</span>
        <span className="muted">{l.sqft.toLocaleString()} sq ft</span>
      </div>
    </li>
  );
}

function ErrorBanner({ error }: { error: ApiError }) {
  const isValidation = error.status === 400 && error.details.length > 0;
  return (
    <div className="banner error" role="alert">
      <strong>{isValidation ? "Please fix these and search again:" : "Something went wrong."}</strong>
      {isValidation ? (
        <ul>{error.details.map((d) => <li key={d.field + d.message}>{d.message}</li>)}</ul>
      ) : (
        <p>{error.message}</p>
      )}
    </div>
  );
}

function Pagination({ data, disabled, onPage }: { data: SearchResponse; disabled: boolean; onPage: (p: number) => void }) {
  const { page, totalPages } = data;
  const pages = Array.from({ length: totalPages }, (_, i) => i + 1);

  return (
    <nav className="pagination" aria-label="Pages">
      <button type="button" className="btn btn-secondary" onClick={() => onPage(page - 1)}
        disabled={disabled || page <= 1}>
        Previous
      </button>
      <div className="pages">
        {pages.map((p) => (
          <button key={p} type="button" onClick={() => onPage(p)} disabled={disabled}
            aria-current={p === page ? "page" : undefined}
            className={`btn page${p === page ? " current" : ""}`}>
            {p}
          </button>
        ))}
      </div>
      <span className="muted page-of">Page {page} of {totalPages}</span>
      <button type="button" className="btn btn-secondary" onClick={() => onPage(page + 1)}
        disabled={disabled || page >= totalPages}>
        Next
      </button>
    </nav>
  );
}
