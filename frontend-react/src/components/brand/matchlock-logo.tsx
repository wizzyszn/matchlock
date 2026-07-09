import logoUrl from "@/assets/g17.svg";
function MatchLockLogo() {
  return (
    <div className="flex  items-center justify-center gap-2">
      <img src={logoUrl} alt="" className="h-6 w-auto object-contain" />
      <span className="text-lg">Matchlock</span>
    </div>
  );
}

export default MatchLockLogo;