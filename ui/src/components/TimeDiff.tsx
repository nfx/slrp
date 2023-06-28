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
