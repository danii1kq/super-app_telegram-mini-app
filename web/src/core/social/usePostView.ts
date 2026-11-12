import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { feedKeys } from './api';
import { useRecordView } from './mutations';

export function usePostView(postId: string, viewerId: string, enabled: boolean) {
  const element = useRef<HTMLElement>(null);
  const attempted = useRef(false);
  const queryClient = useQueryClient();
  const record = useRecordView(postId, viewerId);
  const mutate = record.mutate;

  useEffect(() => {
    const target = element.current;
    if (!target || !enabled || queryClient.getQueryData(feedKeys.viewed(viewerId, postId))) return;
    let timer: number | undefined;
    let observer: IntersectionObserver;
    const clear = () => { window.clearTimeout(timer); timer = undefined; };
    const observe = () => {
      clear();
      observer?.disconnect();
      const header = Math.max(0, document.querySelector('header')?.getBoundingClientRect().bottom ?? 0);
      const footer = Math.max(0, document.querySelector('nav')?.getBoundingClientRect().height ?? 0);
      const visibleHeight = Math.max(1, window.innerHeight - header - footer);
      // Fixed navigation hides content; very tall posts need half the remaining viewport.
      const threshold = Math.min(0.5, visibleHeight / Math.max(1, target.getBoundingClientRect().height) / 2);
      observer = new IntersectionObserver(([entry]) => {
        clear();
        if (!entry?.isIntersecting || entry.intersectionRatio < threshold || document.hidden || attempted.current) return;
        timer = window.setTimeout(() => {
          if (document.hidden || attempted.current) return;
          attempted.current = true;
          mutate(undefined, { onError: () => { attempted.current = false; } });
        }, 1000);
      }, { threshold: [0, threshold], rootMargin: `-${header}px 0px -${footer}px 0px` });
      observer.observe(target);
    };
    const visibility = () => { if (document.hidden) clear(); else observe(); };
    const resize = new ResizeObserver(observe);
    resize.observe(target);
    for (const overlay of document.querySelectorAll('header, nav')) resize.observe(overlay);
    observe();
    document.addEventListener('visibilitychange', visibility);
    window.addEventListener('resize', observe);
    window.addEventListener('online', observe);
    return () => {
      clear(); observer.disconnect(); resize.disconnect();
      document.removeEventListener('visibilitychange', visibility);
      window.removeEventListener('resize', observe);
      window.removeEventListener('online', observe);
    };
  }, [postId, viewerId, enabled, queryClient, mutate]);
  return element;
}
