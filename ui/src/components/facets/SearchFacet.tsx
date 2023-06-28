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
