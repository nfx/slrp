export type CardProps = {
  label: string;
  value: number;
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
