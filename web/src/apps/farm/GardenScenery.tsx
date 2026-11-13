import { useId } from 'react';

export function GardenScenery() {
  const id = useId();
  return <div className="farm-garden-scenery farm-village" aria-hidden="true">
    <svg className="farm-village-art" viewBox="0 0 720 190" fill="none" preserveAspectRatio="xMidYMin slice">
      <defs>
        <linearGradient id={`${id}-sky`} x2="0" y2="1"><stop stopColor="var(--scene-sky)"/><stop offset="1" stopColor="var(--scene-ground)"/></linearGradient>
        <linearGradient id={`${id}-glass`} x1="0" y1="0" x2="1" y2="1"><stop stopColor="var(--scene-glass)" stopOpacity=".9"/><stop offset="1" stopColor="var(--scene-glass)" stopOpacity=".35"/></linearGradient>
      </defs>
      <path d="M0 0h720v190H0Z" fill={`url(#${id}-sky)`}/>
      <g className="farm-village-sun"><circle cx="389" cy="38" r="25" fill="#fff3bd" opacity=".16"/><circle cx="389" cy="38" r="16" fill="#fbe5a7"/><circle cx="383" cy="32" r="4" fill="#fff8d8" opacity=".6"/></g>
      <g className="farm-village-moon"><circle cx="389" cy="38" r="26" fill="#dfdcff" opacity=".08"/><path d="M398 18c-19 2-21 27-3 33-23 5-35-22-17-34 6-4 12-3 20 1Z" fill="#eee4ff"/><path d="m437 21 2 5 5 2-5 2-2 5-2-5-5-2 5-2 2-5Zm-103 8 2 4 4 2-4 2-2 4-2-4-4-2 4-2 2-4Z" fill="#ebdca6"/></g>
      <g className="farm-village-clouds" fill="var(--scene-cloud)" opacity=".45"><path d="M263 30c-9-14-30-10-31 2-12-7-23 2-23 9h77c1-9-11-16-23-11ZM461 56c-7-10-23-8-25 1-12-4-21 3-21 9h64c0-8-9-12-18-10Z"/></g>
      <path d="M0 100c110-67 169 10 269-12 115-24 190-51 299-8 54 21 112 10 152 1v109H0Z" fill="var(--scene-hill)"/>
      <path d="M0 143c133-39 201 22 320-5 159-35 280-6 400-29v81H0Z" fill="var(--scene-ground)"/>
      <path d="M336 83c-35 22 13 39 39 52 38 20 15 39-17 55h79c37-35-13-57-48-74-30-16-32-22-25-33Z" fill="var(--scene-path)" opacity=".7"/>
      <path d="m350 102 9 5m35 26 9 5m-4 34-8 6" stroke="var(--scene-path-light)" strokeWidth="3" strokeLinecap="round"/>
      <g className="farm-village-house">
        <ellipse cx="122" cy="120" rx="75" ry="12" fill="#213c31" opacity=".12"/>
        <path d="M70 64h100v55H70Z" fill="var(--scene-wall)"/>
        <path className="farm-village-roof" d="m62 67 62-43 61 43-10 6-51-36-52 36-10-6Z" fill="var(--scene-roof)"/>
        <path d="m124 24 61 43-10 6-51-36v-13Z" fill="#253932" opacity=".15"/>
        <path d="M70 82h100m-100 17h100" stroke="#ffffff" strokeOpacity=".14" strokeWidth="2"/>
        <path d="M109 118V88a15 15 0 0 1 30 0v30h-30Z" fill="var(--scene-door)"/>
        <path d="M113 118V90a11 11 0 0 1 22 0v28h-22Z" stroke="#ffffff" strokeOpacity=".2" strokeWidth="2"/>
        <circle cx="132" cy="103" r="2" fill="#f0d397"/>
        <rect x="79" y="78" width="20" height="22" rx="3" fill="#fae7ad"/><path d="M89 78v22m-10-11h20" stroke="var(--scene-door)" strokeWidth="2"/>
        <rect x="148" y="78" width="15" height="22" rx="3" fill="#fae7ad"/><path d="M155 78v22m-7-11h15" stroke="var(--scene-door)" strokeWidth="2"/>
        <path d="M146 39V19h14v30" fill="var(--scene-wall)"/><path d="M143 19h20" stroke="var(--scene-roof)" strokeWidth="5" strokeLinecap="round"/>
        <path d="M103 120h41l5 5H98l5-5Z" fill="var(--scene-path-light)"/>
        <path d="M50 115c-5-19 16-26 23-11 7-11 23-4 21 11H50Zm106 3c-3-15 13-24 23-14 10-7 21 2 19 14h-42Z" fill="var(--scene-tree)"/>
      </g>
      <g className="farm-village-greenhouse">
        <ellipse cx="582" cy="128" rx="81" ry="12" fill="#213c31" opacity=".12"/>
        <path d="M514 74 581 39l66 35v52H514V74Z" fill={`url(#${id}-glass)`} stroke="var(--scene-frame)" strokeWidth="4" strokeLinejoin="round"/>
        <path d="M514 74h133M581 39v87m-34-69v69m67-69v69m-100-24h133" stroke="var(--scene-frame)" strokeWidth="3"/>
        <path d="m523 78 16 17m18-31 15 15m19-28 12 12m19 20 16 16" stroke="#ffffff" strokeOpacity=".28" strokeWidth="5" strokeLinecap="round"/>
        <path d="M568 126v-24a13 13 0 0 1 26 0v24" fill="var(--scene-glass)" stroke="var(--scene-frame)" strokeWidth="3"/>
        <g fill="var(--scene-tree)"><path d="M530 121v-12c-16-3-14-15-10-18 13 0 16 10 10 18 1-12 11-19 17-16 1 12-10 19-17 16Zm96 1v-11c-14-2-16-13-12-18 12-2 16 9 12 18 2-13 11-19 18-14-1 11-11 17-18 14Z"/></g>
        <path d="M518 127h127" stroke="var(--scene-frame)" strokeWidth="5" strokeLinecap="round"/>
      </g>
      <g className="farm-village-pond">
        <ellipse cx="214" cy="143" rx="47" ry="17" fill="var(--scene-pond-edge)"/>
        <ellipse cx="214" cy="140" rx="40" ry="12" fill="var(--scene-water)"/>
        <path d="M186 137c10-6 24-7 35-4m-10 14c11 1 20-1 25-4" stroke="#ffffff" strokeOpacity=".35" strokeWidth="2" strokeLinecap="round"/>
        <g className="farm-village-ripple"><ellipse cx="232" cy="137" rx="9" ry="3" stroke="#ffffff" strokeOpacity=".3"/></g>
        <ellipse cx="202" cy="139" rx="9" ry="3" fill="var(--scene-tree)"/><path d="m202 139 4-2 2 4" fill="var(--scene-water)"/>
        <path d="M201 137c-7-1-8-7-4-10l4 5 5-5c4 4 2 9-5 10Z" fill="#ecd0d8"/>
        <path d="M175 132v-17m4 18 2-22" stroke="var(--scene-tree)" strokeWidth="2" strokeLinecap="round"/><path d="M175 119v-7m6 4v-8" stroke="#b9a17b" strokeWidth="4" strokeLinecap="round"/>
      </g>
      <g fill="var(--scene-tree)"><path d="M16 127c-8-20-14-37-6-52 18 2 28 14 23 31 17-13 30-10 32 1-6 17-23 21-49 20Z"/><path d="M671 132c-4-19 3-34 16-37 14 6 13 17 7 25 10-7 22-4 26 7v11l-49-6Z"/></g>
      <g stroke="var(--scene-fence)" strokeWidth="4" strokeLinecap="round"><path d="M0 158h88m-83-13v26m21-26v26m21-26v26m21-26v26m21-26v26m542-11h89m-83-14v26m21-26v26m21-26v26m21-26v26m21-26v26"/></g>
      <g fill="var(--scene-path-light)"><ellipse cx="278" cy="134" rx="9" ry="4" transform="rotate(-15 278 134)"/><ellipse cx="303" cy="145" rx="7" ry="3" transform="rotate(15 303 145)"/><ellipse cx="466" cy="151" rx="9" ry="4" transform="rotate(-8 466 151)"/></g>
      <g fill="#e9d29b"><circle cx="42" cy="127" r="3"/><circle cx="48" cy="121" r="2"/><circle cx="664" cy="140" r="3"/><circle cx="677" cy="147" r="2"/></g>
      <path d="M332 67v21m42-20v20" stroke="var(--scene-fence)" strokeWidth="4"/>
      <rect x="316" y="55" width="77" height="26" rx="5" fill="var(--scene-sign)" stroke="var(--scene-fence)" strokeWidth="2" transform="rotate(-3 354 68)"/>
      <text x="354" y="72" textAnchor="middle" fill="var(--scene-sign-ink)" fontSize="13" fontWeight="650" fontFamily="inherit" transform="rotate(-3 354 68)">мой сад</text>
    </svg>
    <span className="farm-village-edge edge-left"/><span className="farm-village-edge edge-right"/>
  </div>;
}
