export function IconHeader({ icon, col, title }: { icon: string; col?: string; title: string }) {
  const cl = `bi bi-${icon}`;
  return (
    <th className={`col-${col}`}>
      <i className={cl} title={title} />
    </th>
  );
}
