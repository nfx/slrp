import { useEffect, useState, useRef, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import React from "react";
import { http } from "../util";

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
          {total !== undefined && <span className="total">{total} total</span>}
          <input className="form-control form-control-dark w-100 border-secondary" type="text" value={filter} onChange={change} placeholder="Search" aria-label="Search" />
        </div>
        <button className="btn btn-outline-secondary border-secondary" type="button" onClick={togglePause} title={!pause ? "Pause live update" : "Resume live update"}>
          <i className={`bi ${pause ? "bi-play" : "bi-pause"}`} />
        </button>
      </div>
      {failure !== undefined && (
        <div className="alert-danger" role="alert">
          {failure}
        </div>
      )}
    </div>
  );
}
