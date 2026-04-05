
function isStandaloneWebapp(): boolean {
  if (typeof window === 'undefined') return false;

  const nav = window.navigator as Navigator & { standalone?: boolean };

  return nav.standalone === true ||
    window.matchMedia('(display-mode: standalone)').matches;
}

function isMobileBrowser(): boolean {
  if (typeof window === 'undefined') return false;

  const isMobileViewport = window.matchMedia('(pointer: coarse)').matches;
  const ua = navigator.userAgent.toLowerCase();
  const isMobileUA = /android|iphone|ipad|ipod|mobile/.test(ua);

  return isMobileUA || isMobileViewport;
}

export function isMobileNotWebapp(): boolean {  
  return (isMobileBrowser() && !isStandaloneWebapp());
}

let helpOpen = $state(true)

export function isHelpOpen(){
    console.log("helpopen its", helpOpen)
    return helpOpen
}

export function toggleHelpOpen(){
    console.log("shuu")
    helpOpen = !helpOpen
}

export function setHelpOpen(state:boolean){
  helpOpen = state
}
      