import axios from "axios";
import { useEffect, useRef } from "react";

export const http = axios.create({
  baseURL: "/api"
});

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

export function tinyNum(n?: number) {
  if (!n) {
    return 0;
  }
  if (n < 1000) {
    return n;
  } else if (n < 1000000) {
    return (n / 1000).toFixed() + "k";
  }
  return (n / 1000000).toFixed(1) + "m";
}
