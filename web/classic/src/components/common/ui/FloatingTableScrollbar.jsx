/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useRef, useState } from 'react';
import PropTypes from 'prop-types';

const SCROLL_CONTAINER_SELECTORS = [
  '.semi-table-body',
  '.semi-table-container',
  '.semi-table-content',
];

const getScrollableElement = (root) => {
  if (!root) return null;

  for (const selector of SCROLL_CONTAINER_SELECTORS) {
    const candidates = Array.from(root.querySelectorAll(selector));
    const target = candidates.find(
      (element) => element.scrollWidth > element.clientWidth + 1,
    );
    if (target) return target;
  }

  return Array.from(root.querySelectorAll('*')).find((element) => {
    const style = window.getComputedStyle(element);
    const canScrollX = ['auto', 'scroll'].includes(style.overflowX);
    return canScrollX && element.scrollWidth > element.clientWidth + 1;
  });
};

const setScrollElement = (nextElement, currentElement, syncFromTable) => {
  if (currentElement === nextElement) {
    currentElement?.classList.add('floating-table-scrollbar-source');
    return currentElement;
  }

  currentElement?.classList.remove('floating-table-scrollbar-source');
  currentElement?.removeEventListener('scroll', syncFromTable);
  nextElement?.classList.add('floating-table-scrollbar-source');
  nextElement?.addEventListener('scroll', syncFromTable, { passive: true });
  return nextElement || null;
};

const FloatingTableScrollbar = ({ className = '', useParentArea = false }) => {
  const scrollbarAreaRef = useRef(null);
  const barRef = useRef(null);
  const scrollElementRef = useRef(null);
  const syncingRef = useRef(false);
  const frameRef = useRef(null);
  const [barState, setBarState] = useState({
    visible: false,
    offsetLeft: 0,
    width: 0,
    scrollWidth: 0,
  });

  const handleBarScroll = () => {
    if (syncingRef.current) return;
    const bar = barRef.current;
    const scrollElement = scrollElementRef.current;
    if (!bar || !scrollElement) return;
    syncingRef.current = true;
    scrollElement.scrollLeft = bar.scrollLeft;
    syncingRef.current = false;
  };

  useEffect(() => {
    const scrollbarArea = scrollbarAreaRef.current;
    if (!scrollbarArea) return undefined;

    const card = scrollbarArea.closest('.table-scroll-card');
    if (!card) return undefined;

    const syncFromTable = () => {
      if (syncingRef.current) return;
      const bar = barRef.current;
      const scrollElement = scrollElementRef.current;
      if (!bar || !scrollElement) return;
      syncingRef.current = true;
      bar.scrollLeft = scrollElement.scrollLeft;
      syncingRef.current = false;
    };

    const update = () => {
      if (frameRef.current) cancelAnimationFrame(frameRef.current);
      frameRef.current = requestAnimationFrame(() => {
        const scrollElement = getScrollableElement(card);
        scrollElementRef.current = setScrollElement(
          scrollElement,
          scrollElementRef.current,
          syncFromTable,
        );

        if (!scrollElement) {
          setBarState((state) => ({ ...state, visible: false }));
          return;
        }

        const areaRect = scrollbarArea.getBoundingClientRect();
        const elementRect = scrollElement.getBoundingClientRect();
        const hasHorizontalOverflow =
          scrollElement.scrollWidth > scrollElement.clientWidth + 1;

        if (!hasHorizontalOverflow) {
          setBarState((state) => ({ ...state, visible: false }));
          return;
        }

        const offsetLeft = Math.max(elementRect.left - areaRect.left, 0);
        const width = Math.max(scrollElement.clientWidth, 0);

        setBarState({
          visible: width > 0,
          offsetLeft,
          width,
          scrollWidth: scrollElement.scrollWidth,
        });

        const bar = barRef.current;
        if (bar && bar.scrollLeft !== scrollElement.scrollLeft) {
          bar.scrollLeft = scrollElement.scrollLeft;
        }
      });
    };

    const resizeObserver = new ResizeObserver(update);
    resizeObserver.observe(card);
    resizeObserver.observe(scrollbarArea);

    const mutationObserver = new MutationObserver(update);
    mutationObserver.observe(card, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['class', 'style'],
    });

    const bindScrollElement = () => {
      const scrollElement = getScrollableElement(card);
      const previousElement = scrollElementRef.current;
      scrollElementRef.current = setScrollElement(
        scrollElement,
        previousElement,
        syncFromTable,
      );
      if (scrollElement && scrollElement !== previousElement) {
        resizeObserver.observe(scrollElement);
      }
    };

    const handleScroll = () => {
      bindScrollElement();
      update();
    };

    bindScrollElement();
    update();
    window.addEventListener('resize', update);
    window.addEventListener('scroll', handleScroll, true);

    return () => {
      if (frameRef.current) cancelAnimationFrame(frameRef.current);
      resizeObserver.disconnect();
      mutationObserver.disconnect();
      window.removeEventListener('resize', update);
      window.removeEventListener('scroll', handleScroll, true);
      scrollElementRef.current?.classList.remove('floating-table-scrollbar-source');
      scrollElementRef.current?.removeEventListener('scroll', syncFromTable);
    };
  }, []);

  const content = (
    <>
      {barState.visible && (
        <div
          ref={barRef}
          className='floating-table-scrollbar'
          onScroll={handleBarScroll}
          style={{ marginLeft: barState.offsetLeft, width: barState.width }}
        >
          <div style={{ width: barState.scrollWidth, height: 1 }} />
        </div>
      )}
    </>
  );

  if (useParentArea) {
    return (
      <div ref={scrollbarAreaRef} className={className}>
        {content}
      </div>
    );
  }

  return (
    <div ref={scrollbarAreaRef} className={`floating-table-scrollbar-area ${className}`}>
      {content}
    </div>
  );
};

FloatingTableScrollbar.propTypes = {
  className: PropTypes.string,
  useParentArea: PropTypes.bool,
};

export default FloatingTableScrollbar;
