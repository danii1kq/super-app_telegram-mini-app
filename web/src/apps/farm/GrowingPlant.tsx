const stageNames = ['Всходы', 'Набирает силу', 'Появляются плоды'] as const;

export function growthStage(progress: number): 0 | 1 | 2 {
  return progress >= .7 ? 2 : progress >= .35 ? 1 : 0;
}

export function growthLabel(progress: number): string { return stageNames[growthStage(progress)]; }

/** Species keep their own silhouette even before fruit appears. */
export function GrowingPlant({ seed, progress = 0, variant = 0 }: { seed: string; progress?: number; variant?: number }) {
  const stage = growthStage(progress);
  const scale = [.73, .88, 1][stage];
  const lean = [-5, 4, -2, 6, 0, -4, 3, -1][Math.abs(variant) % 8];
  return <g data-growth-stage={stageNames[stage]} transform={`translate(60 103) rotate(${lean}) scale(${scale}) translate(-60 -103)`}>
    {seed === 'tomato' && <>
      <path d="M60 102V44m0 36L38 64m22 2 23-16" stroke="#4c813b" strokeWidth="4" strokeLinecap="round"/>
      <path d="m61 50-4-13 10-9 6 10-5 12m-12 27-12-3-8-10-12-3 9-8 14 5 5 10m15-7 10-15 17-5 2 10-12 6-7 13" fill="#6aa34c"/>
      <path d="m59 91-15-1-10-9 7-8 14 8m9 2 14-10 13 2-5 9-16 5" fill="#86b556"/>
      {stage > 0 && <><path d="m57 55-16-9-4-15 12 5 11 11m4 19 9 11 15 1" stroke="#5b9144" strokeWidth="3" strokeLinecap="round"/><path d="m36 32 1-8 5 6 7-1-4 6 2 6-7-3-5 4 1-10Z" fill="#e6ce65"/></>}
      {stage === 2 && <><circle cx="83" cy="83" r="9" fill="#9fb55a"/><circle cx="40" cy="69" r="7" fill="#d8ba64"/><path d="m77 75 6 3 6-3-3 7-4-2-4 3-1-8Z" fill="#538047"/></>}
    </>}
    {seed === 'pumpkin' && <>
      <path d="M27 96c14-11 19-4 32-12s17-5 30-12M58 86 48 67m20 13 7-22" stroke="#698f3d" strokeWidth="4" strokeLinecap="round"/>
      <path d="m48 75-20-3-7-15 10-14 19-2 12 14-4 15-10 5Z" fill="#76a74b"/><path d="m72 66-13-13 3-17 17-7 17 13-3 16-21 8Z" fill="#659945"/>
      <path d="m46 68-8-16m35 6 5-18" stroke="#b2cc78" strokeWidth="2" strokeLinecap="round"/>
      <path d="M90 76c15-11 18 3 9 5s-8-7-3-9" stroke="#7c9b4d" strokeWidth="2"/>
      {stage > 0 && <><path d="m38 97-14-2-8-12 9-13 16 2 10 13-13 12Z" fill="#8db75d"/><path d="m83 63 2-13 7 8 9-4-4 12-14-3Z" fill="#eac364"/></>}
      {stage === 2 && <><ellipse cx="71" cy="94" rx="18" ry="12" fill="#bcaa53"/><ellipse cx="71" cy="94" rx="9" ry="12" fill="#ceba63"/><path d="M71 82v-8" stroke="#6b833d" strokeWidth="3" strokeLinecap="round"/></>}
    </>}
    {seed === 'carrot' && <>
      <path d="M59 102 47 45m13 55 6-69m-5 69 24-48m-27 48-25-24" stroke="#629b42" strokeWidth="3" strokeLinecap="round"/>
      <path d="m47 60-14-8m15 3 8-13m-6 30-16-6m18 0 8-13m5-5-9-13m10 6 9-13m-9 32-9-10m9 4 12-11m-1 27-5-10m1 19 12-7M43 86l-12-2" stroke="#86b54f" strokeWidth="6" strokeLinecap="round"/>
      {stage > 0 && <path d="m58 103-3-28-12-10m19 37 14-20 12-5m-13 9-1-10" stroke="#9ac462" strokeWidth="4" strokeLinecap="round"/>}
      {stage === 2 && <path d="M52 103c0-12 17-12 17 0" fill="#e29a58"/>}
    </>}
    {seed === 'radish' && <>
      <path d="m60 103-4-29m5 27 19-24m-22 23-19-15" stroke="#a56e62" strokeWidth="4" strokeLinecap="round"/>
      <path d="M54 86C35 75 39 43 52 35c15 9 18 35 2 51Z" fill="#84ad54"/><path d="M66 92c-1-26 15-43 29-40 1 19-8 37-29 40Z" fill="#5e974d"/><path d="M46 96C22 92 20 76 24 65c18 1 29 15 22 31Z" fill="#9ac36c"/>
      <path d="m54 78-2-29m20 34 16-24m-46 30L29 74" stroke="#caDB98" strokeWidth="2" strokeLinecap="round"/>
      {stage > 0 && <path d="M59 82c-4-24 5-42 18-44 7 16 3 33-18 44Z" fill="#76a45a"/>}
      {stage === 2 && <path d="M50 103c-4-15 22-15 21 0" fill="#d67a92"/>}
    </>}
    {seed === 'strawberry' && <>
      <path d="M58 102 48 74m12 28 17-31m-17 30L30 86" stroke="#598d48" strokeWidth="3"/>
      <g fill="#6a9e53"><path d="M47 79c-19-4-23-17-16-22 11-7 20 4 16 22Z"/><path d="M48 79c-12-15-10-28 0-30 11 0 13 16 0 30Z"/><path d="M50 79c-1-20 11-28 19-20 6 9-5 20-19 20Z"/></g>
      <g fill="#85ad61"><path d="M77 80c-20-5-20-19-12-22 12-3 16 8 12 22Z"/><path d="M78 80c-12-18-6-29 4-27 10 4 7 17-4 27Z"/><path d="M79 81c2-21 16-22 20-13 2 11-9 15-20 13Z"/></g>
      {stage > 0 && <><path d="m58 98 5-16" stroke="#5e8e4b" strokeWidth="2"/>{[0,72,144,216,288].map(angle=><ellipse key={angle} cx="64" cy="75" rx="4" ry="7" transform={`rotate(${angle} 64 81)`} fill="#fff4d7"/>)}<circle cx="64" cy="81" r="4" fill="#e4c05b"/></>}
      {stage === 2 && <><path d="M32 84c-13-4-13 6-6 15 4 5 12-3 13-8 1-5-3-9-7-7Z" fill="#d99b82"/><path d="m27 84 5 4 6-4" stroke="#5f8c4c" strokeWidth="3"/></>}
    </>}
    {seed === 'blueberry' && <>
      <path d="M59 102V41m0 39L35 58m24 12 22-25m-22 45 29-15" stroke="#8b7e5d" strokeWidth="3.5" strokeLinecap="round"/>
      <g fill="#698e63"><ellipse cx="37" cy="54" rx="7" ry="15" transform="rotate(-40 37 54)"/><ellipse cx="79" cy="47" rx="8" ry="15" transform="rotate(36 79 47)"/><ellipse cx="60" cy="37" rx="8" ry="16"/></g>
      <g fill="#8dab73"><ellipse cx="89" cy="70" rx="8" ry="15" transform="rotate(52 89 70)"/><ellipse cx="43" cy="82" rx="7" ry="14" transform="rotate(-55 43 82)"/></g>
      {stage > 0 && <><path d="m55 61-12-19m19 26 13 12" stroke="#938164" strokeWidth="2"/><ellipse cx="43" cy="40" rx="6" ry="12" fill="#779e70"/></>}
      {stage === 2 && <g fill="#9aaf99"><circle cx="80" cy="86" r="6"/><circle cx="91" cy="84" r="6"/><circle cx="85" cy="94" r="6"/><circle cx="35" cy="70" r="5"/></g>}
    </>}
    {seed === 'dragonfruit' && <>
      <path d="M53 103 48 43l11-17 12 19-4 58" fill="#6ba48a"/><path d="m51 84-16-7-7-27 9-10 10 13 5 31Z" fill="#86b99a"/><path d="m68 90 17-13 8-26-8-9-12 15-5 33Z" fill="#55937c"/>
      <path d="m59 39 1 58m-23-42 7 20m39-18-9 24" stroke="#b1d3aa" strokeWidth="2" strokeLinecap="round"/>
      {[[49,53],[69,64],[53,77],[71,89],[30,58],[87,65]].map(([x,y],i)=><path key={i} d={`m${x} ${y} -3 -3`} stroke="#e0dfb5" strokeWidth="1.5" strokeLinecap="round"/>)}
      {stage > 0 && <path d="m57 31-2-10 8-6 8 10-6 9" fill="#b68d9a"/>}
      {stage === 2 && <><path d="M78 70c-4-12 5-17 11-13s7 14-2 20Z" fill="#ce85a4"/><path d="m78 62 6 3 6-4m-11 11 5-1 5 4" stroke="#95ba7f" strokeWidth="2"/></>}
    </>}
    {seed === 'cucumber' && <>
      <path d="M59 103c-14-25 24-33 10-55-8-14 9-25 15-15 5 8-8 12-8 4" stroke="#6b9d51" strokeWidth="3" strokeLinecap="round"/>
      <path d="m49 94-18-5-7-19 13-12 21 11 1 15-10 10Zm24-20-13-13 4-20 20-6 15 16-8 16-18 7Z" fill="#79a554"/>
      <path d="M41 70 49 85m29-36-5 16" stroke="#b9d18b" strokeWidth="2"/>
      {stage > 0 && <><path d="m62 74 23 9m-5-5c12-9 18 0 10 5" stroke="#83a557" strokeWidth="2"/><path d="m63 48 4-10 5 7 9-1-6 7 1 8-8-5-7 3 2-9Z" fill="#efd17b"/></>}
      {stage === 2 && <rect x="78" y="80" width="12" height="23" rx="6" transform="rotate(17 78 80)" fill="#88a65a"/>}
    </>}
    {seed === 'corn' && <>
      <path d="M60 103V30" stroke="#699745" strokeWidth="5" strokeLinecap="round"/>
      <path d="M59 91C37 85 27 70 22 47c23 10 37 25 37 44Z" fill="#87b357"/><path d="M62 75c4-28 18-44 38-48-9 21-20 35-38 48Z" fill="#6d9e49"/><path d="M60 56C45 42 40 26 43 13c15 10 19 25 17 43Z" fill="#99bf64"/>
      <path d="m31 60 24 26m15-26 19-22" stroke="#b9cf83" strokeWidth="2" strokeLinecap="round"/>
      {stage > 0 && <path d="M62 99c9-26 21-39 40-43-10 25-23 35-40 43Z" fill="#92b85d"/>}
      {stage === 2 && <><path d="m64 77 9-18 10-3-1 14-16 14" fill="#c5c170"/><path d="M60 31V14m0 9-8-9m8 4 9-9" stroke="#d0bd79" strokeWidth="2" strokeLinecap="round"/></>}
    </>}
    {seed === 'sunflower' && <>
      <path d="M60 103V35" stroke="#68974c" strokeWidth="5" strokeLinecap="round"/>
      <path d="M58 82c-25 3-35-12-32-28 24-2 34 12 32 28Z" fill="#80ab53"/><path d="M63 69c0-24 15-33 34-25-1 21-17 28-34 25Z" fill="#689c50"/>
      <path d="m55 77-18-15m29 1 21-11" stroke="#b4cb7b" strokeWidth="2" strokeLinecap="round"/>
      <path d="M60 49c-19-3-19-23-2-31 20 4 23 24 2 31Z" fill="#a2bb67"/>
      {stage > 0 && <><path d="m45 33 15 12 16-14m-15 13-3-19" stroke="#6b974d" strokeWidth="2"/><path d="M58 98c-13-11-22-10-32-7 4 11 17 15 32 7Z" fill="#92b45c"/></>}
      {stage === 2 && <><circle cx="60" cy="33" r="11" fill="#cbb65d"/><circle cx="60" cy="33" r="6" fill="#987c4d"/></>}
    </>}
    {seed === 'watermelon' && <>
      <path d="M24 99c15-15 36-2 42-17s19-10 29-14M51 94 42 73m26 10 9-28" stroke="#5f924a" strokeWidth="3.5" strokeLinecap="round"/>
      <path d="m41 80-11-4-8-12 9-6 3-14 11 10 14-2-3 15-15 13Z" fill="#6a9f56"/><path d="m76 66-12-7 1-15 11-4 8-15 7 16 13 8-9 13-19 4Z" fill="#85b167"/>
      <path d="m42 72-6-15m43 1 4-17" stroke="#bed39b" strokeWidth="2"/>
      {stage > 0 && <path d="M91 71c17-11 19 4 8 5s-4-10 0-11" stroke="#7fa159" strokeWidth="2" strokeLinecap="round"/>}
      {stage === 2 && <><ellipse cx="62" cy="98" rx="18" ry="11" fill="#a1bd70"/><path d="M51 90c-5 5-5 11 0 16m10-19c-4 7-4 13 0 21m10-17c4 5 4 10 0 15" stroke="#6d9a53" strokeWidth="3"/></>}
    </>}
    {seed === 'moonflower' && <>
      <path d="M60 103c-6-18 7-31 0-56" stroke="#85a5a3" strokeWidth="4" strokeLinecap="round"/>
      <path d="M58 86C35 83 29 67 38 56c19 3 25 14 20 30Z" fill="#97b9ad"/><path d="M63 75c-1-24 13-34 29-27-1 19-11 29-29 27Z" fill="#6f9ca0"/>
      <path d="M60 53C39 45 48 21 61 15c16 13 18 31-1 38Z" fill="#b4b2cd"/><path d="m60 22 1 24m-7 3 7-6 8 3" stroke="#ded7ed" strokeWidth="2" strokeLinecap="round"/>
      {stage > 0 && <path d="M58 98c-13-9-23-9-31-3 6 10 20 12 31 3Z" fill="#b0c6b2"/>}
      {stage === 2 && <><path d="M61 52c-15-2-21-13-16-24l16 10 16-10c5 11-1 22-16 24Z" fill="#e1d5ef"/><circle cx="61" cy="42" r="5" fill="#f3e2ab"/></>}
    </>}
    {seed === 'pepper' && <>
      <path d="M59 104V50m0 30L39 64m20 5 22-21" stroke="#648e4b" strokeWidth="4" strokeLinecap="round"/>
      <path d="M57 69C35 66 29 53 34 43c18 1 26 12 23 26Zm6-9c-1-23 12-36 26-32 0 18-10 32-26 32ZM56 91C35 91 24 80 29 71c16-1 27 8 27 20Z" fill="#85ad60"/>
      {stage > 0 && <><path d="M63 88c2-20 16-28 28-20-4 16-15 21-28 20Z" fill="#a2bf78"/><path d="m39 49 2-8 5 5 7-1-3 7 3 6-8-1-5 4-1-8-6-4 6-1Z" fill="#fff2d8"/><circle cx="43" cy="51" r="3" fill="#d9c879"/></>}
      {stage === 2 && <><path d="M74 73c-10-5-14 6-9 17 3 8 14 10 18 0 5-11 0-22-9-17Z" fill="#c7b76c"/><path d="m73 73 2-7" stroke="#688e4c" strokeWidth="3"/></>}
    </>}
    {seed === 'pear' && <>
      <path d="M60 104V36m0 43L39 59m21 7 21-23" stroke="#9b835d" strokeWidth="4" strokeLinecap="round"/>
      <g fill="#86a864"><ellipse cx="38" cy="55" rx="9" ry="16" transform="rotate(-43 38 55)"/><ellipse cx="78" cy="43" rx="10" ry="16" transform="rotate(35 78 43)"/><ellipse cx="59" cy="34" rx="9" ry="17"/></g>
      <path d="M58 91C35 89 29 74 36 68c15 1 24 11 22 23Z" fill="#a3bd7d"/>
      {stage > 0 && <><ellipse cx="86" cy="77" rx="9" ry="15" transform="rotate(42 86 77)" fill="#94b16e"/><path d="m39 42 2-7 5 5 7-1-3 6 3 5-7-1-4 5-2-7-6-2 5-3Z" fill="#faf0db"/></>}
      {stage === 2 && <><path d="M73 79c-6 0-3 8-8 13-8 9-1 14 7 14 11 0 13-8 7-15-3-5 0-12-6-12Z" fill="#c1c47d"/><path d="m73 80 1-6" stroke="#8c8056" strokeWidth="2"/></>}
    </>}
    {seed === 'lavender' && <>
      <path d="M57 104 44 53m14 51 5-68m-3 69 23-45" stroke="#8ca47b" strokeWidth="3" strokeLinecap="round"/>
      <path d="M54 91C33 87 30 75 34 69c13 3 21 11 20 22Zm8 4c2-16 14-24 26-23-2 14-11 23-26 23ZM56 76C42 72 39 61 44 56c11 4 14 12 12 20Z" fill="#a7bd90"/>
      {stage > 0 && <><path d="M62 74c4-15 11-24 22-23-1 13-10 23-22 23Z" fill="#b1c6a2"/><ellipse cx="63" cy="33" rx="4" ry="9" fill="#aba7bd"/><ellipse cx="43" cy="51" rx="4" ry="8" fill="#aba7bd"/></>}
      {stage === 2 && <g fill="#b3a1cc">{[[62,29],[63,39],[63,48],[42,48],[44,57],[84,55],[81,64]].map(([x,y],i)=><ellipse key={i} cx={x} cy={y} rx="5" ry="7"/>)}</g>}
    </>}
    {seed === 'lotus' && <>
      <ellipse cx="59" cy="99" rx="31" ry="10" fill="#8eafa0"/>
      <path d="m59 99 18-6-6 9" fill="#c7d3b3"/><path d="M59 99c-2-12 5-22 2-38" stroke="#83a790" strokeWidth="3" strokeLinecap="round"/>
      <path d="M60 74C43 65 49 49 60 40c14 10 18 25 0 34Z" fill="#a8c1b1"/>
      {stage > 0 && <><ellipse cx="33" cy="86" rx="20" ry="7" fill="#aac4ae"/><path d="M61 68C45 60 53 41 62 33c12 12 14 27-1 35Z" fill="#d0bfc8"/></>}
      {stage === 2 && <><path d="M61 70C41 67 37 54 44 42l18 13 15-16c10 15 3 28-16 31Z" fill="#e7cbd7"/><path d="M61 66c-8-10-4-21 1-29 6 9 11 19-1 29Z" fill="#f2dfe4"/></>}
    </>}
    {seed === 'mango' && <>
      <path d="M60 104V36m0 39L38 54m22 3 22-18" stroke="#8c8961" strokeWidth="4" strokeLinecap="round"/>
      <path d="M58 67C33 65 25 48 29 37c18 5 29 15 29 30Zm6-11c1-24 12-37 28-34-4 20-15 32-28 34Z" fill="#7caa75"/><path d="M59 47C47 39 49 21 59 13c10 11 12 22 0 34Z" fill="#bdad83"/>
      {stage > 0 && <><path d="M62 88c9-19 24-24 35-18-7 13-21 21-35 18Z" fill="#a0bf8b"/><path d="m43 54-8-15m0 0-8-2m8 2 3-9" stroke="#9b9f6c" strokeWidth="2"/>{[[27,36],[35,33],[39,29]].map(([x,y],i)=><circle key={i} cx={x} cy={y} r="3" fill="#e1cba2"/>)}</>}
      {stage === 2 && <><path d="M79 74c-12-6-23 12-18 24 5 10 16 9 23-3 6-11 7-17-5-21Z" fill="#a4b47b"/><path d="m79 74 2-8" stroke="#8c8961" strokeWidth="2"/></>}
    </>}
    {seed === 'starfruit' && <>
      <path d="M59 104V38m0 24L33 51m26 31 30-17" stroke="#839666" strokeWidth="3.5" strokeLinecap="round"/>
      <g fill="#8fad74">{[[59,34,0],[46,54,-50],[31,44,-50],[73,74,45],[88,61,45]].map(([x,y,angle],i)=><ellipse key={i} cx={x} cy={y} rx="7" ry="13" transform={`rotate(${angle} ${x} ${y})`}/>)}</g>
      {stage > 0 && <><path d="M57 93 32 83m27-40 18-9" stroke="#8a9e70" strokeWidth="2"/><ellipse cx="28" cy="79" rx="7" ry="13" transform="rotate(-55 28 79)" fill="#adc392"/><path d="m81 32 2-5 3 5 6 1-4 4 1 6-6-3-5 3 1-6-4-4 6-1Z" fill="#dac2d0"/></>}
      {stage === 2 && <path d="m47 77 4 9 10 1-7 7 1 11-9-5-9 4 2-10-7-8 11-1 4-8Z" fill="#c7c58a"/>}
    </>}
  </g>;
}
