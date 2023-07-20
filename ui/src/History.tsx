import { useState } from "react";
import "./History.css";
import { IconHeader } from "./components/IconHeader";
import { LiveFilter } from "./components/LiveFilter";
import { Facet, QueryFacets } from "./components/facets/QueryFacet";
import { useTitle } from "./util";

function convertSize(bytes: number) {
  if (bytes < 1024) {
    return `${bytes.toFixed()}b`;
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed()}kb`;
  }
  return `${(bytes / 1024 / 1024).toFixed()}mb`;
}

type FilteredRequest = {
  ID: number;
  Serial: number;
  Attempt: number;
  Ts: string;
  Method: string;
  URL: string;
  StatusCode: number;
  Status: string;
  Proxy: string;
  Appeared: number;
  Size: number;
  Took: number;
};

function Request(history: FilteredRequest) {
  const pos = history.URL.indexOf("/", 9);
  const path = history.URL.substring(pos);

  let color = "text-muted";
  if (history.StatusCode < 300) {
    color = "text-success";
  } else if (history.StatusCode < 500) {
    color = "text-warning";
  }
  // TODO: add links in backend
  return (
    <tr className="list-group-item-action">
      <td className="text-muted">
        <small>{new Date(history.Ts).toLocaleTimeString()}</small>
      </td>
      <td>
        <span className="request">
          {history.Method}{" "}
          <a className="app-link" href={`http://localhost:8089/api/history/${history.ID}?format=text`} rel="noreferrer" target="_blank">
            <abbr title={history.URL}>{path}</abbr>
          </a>
          <sup>
            <a className="text-muted" href={`/history?filter=Serial:${history.Serial}`}>
              {history.Serial}
            </a>
          </sup>
        </span>
      </td>
      <td className={color}>
        {history.StatusCode === 200 ? 200 : <abbr title={history.Status}>{history.StatusCode}</abbr>} <sup>{history.Attempt}</sup>
      </td>
      <td className="text-muted proxy">
        <a className="link-primary app-link" href={`/history?filter=Proxy:"${Proxy}"`}>
          {history.Proxy}
        </a>{" "}
        <sup>{history.Appeared}</sup>
      </td>
      <td className="size">{convertSize(history.Size)}</td>
      <td className="took">{history.Took}s</td>
    </tr>
  );
}

export default function History() {
  useTitle("History");
  const [result, setResult] = useState<{ facets: Facet[]; Records?: FilteredRequest[] }>();
  return (
    <div id="history-table" className="card history table-responsive">
      <LiveFilter endpoint="/history" onUpdate={setResult} minDelay={2000} />
      {result && (
        <div>
          <QueryFacets endpoint="/history" {...result} />
          <table className="table text-start table-sm">
            <thead>
              <tr className="text-uppercase text-muted">
                <th></th>
                <IconHeader icon="filetype-raw" title="Click on link to get pretty dump. Click on number to filter by serial." />
                <IconHeader icon="123" title="HTTP status code" />
                <IconHeader icon="link proxy" title="Proxy used" />
                <IconHeader icon="arrow-left-right size" title="Size" />
                <IconHeader icon="hourglass-bottom took" title="Proxy used" />
              </tr>
            </thead>
            <tbody>{result.Records && result.Records.map(r => <Request key={r.ID} {...r} />)}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}
