import axios from "axios";
import { useEffect, useState, useRef, Component, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import React from "react";

export const http = axios.create({
  baseURL: "/api"
});

export class ErrorBoundary extends Component<{ children: React.ReactNode }> {
  state = { error: false, errorMessage: "" };

  static getDerivedStateFromError(error: any) {
    return { error: true, errorMessage: error.toString() };
  }

  componentDidCatch(error: any) {
    console.error(error);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="alert alert-danger" role="alert">
          <h4 className="alert-heading">Failed</h4>
          {this.state.errorMessage}
        </div>
      );
    }
    return this.props.children;
  }
}

type TimeDiffProps = {
  ts: number;
  title: string;
};

export function TimeDiff({ ts, title }: TimeDiffProps) {
  const now = new Date().getTime();
  const it = new Date(ts).getTime();
  let elapsed = Math.abs((now - it) / 1000);
  let value = "?";
  if (elapsed < 60) {
    value = `${elapsed.toFixed()}s`;
  } else if (elapsed < 60 * 60) {
    value = `${(elapsed / 60).toFixed()}m`;
  } else if (elapsed < 60 * 60 * 24) {
    value = `${(elapsed / 60 / 60).toFixed()}h`;
  } else if (elapsed < 60 * 60 * 24 * 7) {
    value = `${(elapsed / 60 / 60 / 24).toFixed()}d`;
  } else if (elapsed < 60 * 60 * 24 * 30) {
    value = `${(elapsed / 60 / 60 / 24 / 7).toFixed()}w`;
  }
  if (it > now) {
    return (
      <sup className="text-muted" title={title}>
        in {value}
      </sup>
    );
  }
  return (
    <sup className="text-muted" title={title}>
      {value} ago
    </sup>
  );
}

export function IconHeader({ icon, col, title }: { icon: string; col?: string; title: string }) {
  const cl = `bi bi-${icon}`;
  return (
    <th className={`col-${col}`}>
      <i className={cl} title={title} />
    </th>
  );
}

export type CardProps = {
  label: string;
  value: string;
  increment?: number;
};

export function Card({ label, value, increment = 0 }: CardProps) {
  return (
    <div className="col">
      <div className="card mb-3">
        <div className="card-body">
          <div className="row align-items-center gx-0">
            <div className="col">
              <h6 className="text-uppercase text-muted mb-2">{label}</h6>
              <span className="h2 mb-0 ">{value}</span>
              {increment > 0 && <span className="badge bg-success ms-2">+{increment}</span>}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

type LiveFilterProps = {
  endpoint: string;
  onUpdate: (data: any) => void;
  minDelay?: number;
};

export function LiveFilter({ endpoint, onUpdate, minDelay = 5000 }: LiveFilterProps) {
  const savedCallback = useRef<number>();
  const [pause, setPause] = useState(false);
  const [total, setTotal] = useState();
  const [failure, setFailure] = useState<number | undefined>();
  const [searchParams, setSearchParams] = useSearchParams();

  let doFilter = useCallback(() => {
    clearTimeout(savedCallback.current);
    savedCallback.current = setTimeout(() => {
      http
        .get(endpoint, { params: searchParams })
        .then(response => {
          if (pause) {
            clearTimeout(savedCallback.current);
            return;
          }
          setTotal(response.data.Total);
          onUpdate(response.data);
          setFailure(undefined);
          savedCallback.current = setTimeout(doFilter, minDelay);
        })
        .catch(err => {
          if (err.isAxiosError) {
            setFailure(err.response.data.Message);
          }
          clearTimeout(savedCallback.current);
          return false;
        });
    }, 500);
  }, [pause, savedCallback, searchParams, endpoint, minDelay, onUpdate]);

  useEffect(() => {
    doFilter();
    return () => clearTimeout(savedCallback.current);
  }, [savedCallback, doFilter]);

  let filter = searchParams.get("filter") || "";
  let change = (event: React.ChangeEvent<HTMLInputElement>) => {
    filter = event.target.value;
    setSearchParams(filter === "" ? {} : { filter });
    doFilter();
  };

  let togglePause = () => {
    setPause(!pause);
    if (pause) {
      doFilter();
    } else {
      clearTimeout(savedCallback.current);
    }
  };

  return (
    <div className="search-filter">
      <div className="input-group">
        <div>
          {total && <span className="total">{total} total</span>}
          <input className="form-control form-control-dark w-100 border-secondary" type="text" value={filter} onChange={change} placeholder="Search" aria-label="Search" />
        </div>
        <button className="btn btn-outline-secondary border-secondary" type="button" onClick={togglePause} title={!pause ? "Pause live update" : "Resume live update"}>
          <i className={`bi ${pause ? "bi-play" : "bi-pause"}`} />
        </button>
      </div>
      {failure && (
        <div className="alert-danger" role="alert">
          {failure}
        </div>
      )}
    </div>
  );
}

function FilterableFacet({ Name, Value, link }: { Name: string; Value: string; link?: string }) {
  const short = Name.length > 32 ? `${Name.substring(0, 32)}...` : Name;
  return (
    <li>
      {link ? (
        <a className="link-primary app-link" href={link.replace("$", Name)}>
          {short}
        </a>
      ) : (
        short
      )}{" "}
      <sup>{Value}</sup>
    </li>
  );
}

export function SearchFacet({ name, items, link }: { name: string; items: { Name: string; Value: string }[]; link?: string }) {
  let result: any[] = [];
  if (items.length > 1) {
    result.push(
      <div key={name} className="search-facet">
        <strong>{name}</strong>
        <ul>
          {items.map(f => (
            <FilterableFacet key={f.Name} link={link} {...f} />
          ))}
        </ul>
      </div>
    );
  }
  return result;
}

function QueryFacetFilter({ Name, Value, Filter, endpoint }: { Name: string; Value: string; Filter: string; endpoint: string }) {
  const short = Name.length > 32 ? `${Name.substring(0, 32)}...` : Name;
  const link = Name !== "n/a" ? `${endpoint}?filter=${Filter}` : undefined;
  return (
    <li>
      {link ? (
        short
      ) : (
        <a className="link-primary app-link" href={link}>
          {short}
        </a>
      )}{" "}
      <sup>{Value}</sup>
    </li>
  );
}

function QueryFacet({ Name, Top, endpoint }: { Name: string; Top: { Name: string; Value: string; Filter: string }[]; endpoint: string }) {
  if (Top.length > 1) {
    return (
      <div key={Name} className="search-facet">
        <strong>{Name}</strong>
        <ul>
          {Top.map(f => (
            <QueryFacetFilter key={f.Name} endpoint={endpoint} {...f} />
          ))}
        </ul>
      </div>
    );
  }
  return <></>;
}

export type Top = { Name: string; Value: string; Filter: string }[];

export type Facet = { Name: string; Top: Top };

export function QueryFacets({ Facets, endpoint }: { Facets?: Facet[]; endpoint: string }) {
  return <>{Facets && Facets.map(f => <QueryFacet key={f.Name} endpoint={endpoint} {...f} />)}</>;
}

export function useInterval(callback: () => any, delay?: number) {
  // https://overreacted.io/making-setinterval-declarative-with-react-hooks/
  const savedCallback = useRef<() => number>();
  useEffect(() => {
    savedCallback.current = callback;
  });
  useEffect(() => {
    function tick() {
      if (savedCallback.current) {
        savedCallback.current();
      }
    }
    if (delay !== undefined) {
      let id = setInterval(tick, delay);
      return () => clearInterval(id);
    }
  }, [delay]);
}

export function useTitle(title: string) {
  useEffect(() => {
    const prev = document.title;
    document.title = `${title} - slrp`;
    return () => {
      document.title = prev;
    };
  }, [title]);
}
