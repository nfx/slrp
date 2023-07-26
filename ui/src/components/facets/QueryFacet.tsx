function QueryFacetFilter({ Name, Value, Filter, endpoint }: { Name: string; Value: string; Filter: string; endpoint: string }) {
  const short = Name.length > 32 ? `${Name.substring(0, 32)}...` : Name;
  const link = Name !== "n/a" ? `${endpoint}?filter=${Filter}` : undefined;
  return (
    <li>
      {link ? (
        <a className="link-primary app-link" href={link}>
          {short}
        </a>
      ) : (
        short
      )}{" "}
      <sup>{Value}</sup>
    </li>
  );
}

function QueryFacet({ Name, Top, endpoint }: { Name: string; Top: { Name: string; Value: string; Filter: string }[]; endpoint: string }) {
  return Top.length == 0 ? (
    <></>
  ) : (
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

export type Facet = { Name: string; Top: { Name: string; Value: string; Filter: string }[] };

export function QueryFacets({ Facets, endpoint }: { Facets?: Facet[]; endpoint: string }) {
  return <>{Facets && Facets.map(f => <QueryFacet key={f.Name} endpoint={endpoint} {...f} />)}</>;
}
