import axios from 'axios'
import {useEffect, useState, useRef, Component} from 'react';
import { useSearchParams } from 'react-router-dom';

export const http = axios.create({
	baseURL: "/api",
});

export class ErrorBoundary extends Component {
	state = { error: false, errorMessage: "" };

	static getDerivedStateFromError(error) {
		return { error: true, errorMessage: error.toString() }
	};

	componentDidCatch(error, _) {
		console.error(error)
	}

	render() {
		if (this.state.error) {
			return <div className="alert alert-danger" role="alert">
				<h4 className="alert-heading">Failed</h4>
				{this.state.errorMessage}
			</div>
		}
		return this.props.children
	}
}

export function TimeDiff({ts, title}) {
  const now = new Date()
  const it = new Date(ts)
  let elapsed = Math.abs((now - it) / 1000)
  let value = '?'
  if (elapsed < 60) {
    value = `${elapsed.toFixed()}s`
  } else if (elapsed < 60*60) {
    value = `${(elapsed/60).toFixed()}m`
  } else if (elapsed < 60*60*24) {
    value = `${(elapsed/60/60).toFixed()}h`
  } else if (elapsed < 60*60*24*7) {
    value = `${(elapsed/60/60/24).toFixed()}d`
  } else if (elapsed < 60*60*24*30) {
    value = `${(elapsed/60/60/24/7).toFixed()}w`
  }
	if (it > now) {
		return <sup className='text-muted' title={title}>in {value}</sup>
	}
  return <sup className='text-muted' title={title}>{value} ago</sup>
}

export function IconHeader({ icon, col, title }) {
  const cl = `bi bi-${icon}`
  return <th className={`col-${col}`}><i className={cl} title={title} /></th>
}

export function Card({ label, value, increment = 0 }) {
	return (
		<div className="col">
			<div className="card mb-3">
				<div className="card-body">
					<div className="row align-items-center gx-0">
						<div className="col">
							<h6 className="text-uppercase text-muted mb-2">
								{label}
							</h6>
							<span className="h2 mb-0 ">
								{value}
							</span>
							{increment > 0 &&
								<span className="badge bg-success ms-2">+{increment}</span>}
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}

export function LiveFilter({ endpoint, onUpdate, minDelay = 5000 }) {
  const savedCallback = useRef();
  const [failure, setFailure] = useState(null);
  const [searchParams, setSearchParams] = useSearchParams();

  let doFilter = () => {
    clearTimeout(savedCallback.current)
    savedCallback.current = setTimeout(() => {
      http.get(endpoint, { params: searchParams }).then(response => {
        onUpdate(response.data)
        setFailure(null)
        savedCallback.current = setTimeout(doFilter, minDelay)
      }).catch(err => {
        if (err.isAxiosError) {
          setFailure(err.response.data.Message)
        }
        clearTimeout(savedCallback.current)
        return false
      })
    }, 500)
  }

  useEffect(() => {
    doFilter()
    return () => clearTimeout(savedCallback.current)
  }, [searchParams]);

  let filter = searchParams.get("filter") || ""
  let change = event => {
    filter = event.target.value
    setSearchParams(filter == "" ? {} : { filter })
    doFilter()
  }
  return <div>
    <input className="form-control form-control-dark w-100" type="text"
      value={filter} onChange={change} placeholder="Search" aria-label="Search" />
    {failure != null && <div className="alert-danger" role="alert">
      {failure}
    </div>}
  </div>
}

export function useInterval(callback, delay) {
	// https://overreacted.io/making-setinterval-declarative-with-react-hooks/
	const savedCallback = useRef();
	useEffect(() => {
		savedCallback.current = callback;
	});
	useEffect(() => {
		function tick() {
			savedCallback.current();
		}
		if (delay !== null) {
			let id = setInterval(tick, delay);
			return () => clearInterval(id);
		}
	}, [delay]);
}

export function useTitle(title) {
  useEffect(() => {
    const prev = document.title;
    document.title = `${title} - slrp`;
    return () => {
      document.title = prev;
    };
  }, []);
}
