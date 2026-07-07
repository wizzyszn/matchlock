/**
 * Maps TxLINE / sportsbook team labels to Flagpedia ISO codes (flagcdn.com).
 * Flagpedia covers countries and subdivisions (e.g. gb-eng); not club crests.
 */

const TEAM_ALIASES: Record<string, string> = {
  // A
  afghanistan: 'af',
  albania: 'al',
  algeria: 'dz',
  andorra: 'ad',
  angola: 'ao',
  'antigua and barbuda': 'ag',
  argentina: 'ar',
  armenia: 'am',
  australia: 'au',
  austria: 'at',
  azerbaijan: 'az',

  // B
  bahamas: 'bs',
  bahrain: 'bh',
  bangladesh: 'bd',
  barbados: 'bb',
  belarus: 'by',
  belgium: 'be',
  belize: 'bz',
  benin: 'bj',
  bhutan: 'bt',
  bolivia: 'bo',
  'bosnia and herzegovina': 'ba',
  'bosnia & herzegovina': 'ba',
  botswana: 'bw',
  brazil: 'br',
  brunei: 'bn',
  bulgaria: 'bg',
  'burkina faso': 'bf',
  burundi: 'bi',

  // C
  cambodia: 'kh',
  cameroon: 'cm',
  canada: 'ca',
  'cape verde': 'cv',
  'central african republic': 'cf',
  chad: 'td',
  chile: 'cl',
  china: 'cn',
  'chinese taipei': 'tw',
  colombia: 'co',
  comoros: 'km',
  congo: 'cg',
  'costa rica': 'cr',
  'cote divoire': 'ci',
  "cote d'ivoire": 'ci',
  'ivory coast': 'ci',
  croatia: 'hr',
  cuba: 'cu',
  curacao: 'cw',
  cyprus: 'cy',
  'czech republic': 'cz',
  czechia: 'cz',

  // D
  denmark: 'dk',
  djibouti: 'dj',
  dominica: 'dm',
  'dominican republic': 'do',
  'dr congo': 'cd',
  'democratic republic of the congo': 'cd',

  // E
  ecuador: 'ec',
  egypt: 'eg',
  'el salvador': 'sv',
  england: 'gb-eng',
  'equatorial guinea': 'gq',
  eritrea: 'er',
  estonia: 'ee',
  eswatini: 'sz',
  ethiopia: 'et',

  // F
  fiji: 'fj',
  finland: 'fi',
  france: 'fr',

  // G
  gabon: 'ga',
  gambia: 'gm',
  georgia: 'ge',
  germany: 'de',
  ghana: 'gh',
  greece: 'gr',
  grenada: 'gd',
  guatemala: 'gt',
  guinea: 'gn',
  'guinea-bissau': 'gw',
  guyana: 'gy',

  // H
  haiti: 'ht',
  honduras: 'hn',
  'hong kong': 'hk',
  hungary: 'hu',

  // I
  iceland: 'is',
  india: 'in',
  indonesia: 'id',
  iran: 'ir',
  iraq: 'iq',
  ireland: 'ie',
  'republic of ireland': 'ie',
  israel: 'il',
  italy: 'it',

  // J
  jamaica: 'jm',
  japan: 'jp',
  jordan: 'jo',

  // K
  kazakhstan: 'kz',
  kenya: 'ke',
  kosovo: 'xk',
  kuwait: 'kw',
  kyrgyzstan: 'kg',
  'south korea': 'kr',
  'korea republic': 'kr',
  'korea rep': 'kr',
  'north korea': 'kp',

  // L
  laos: 'la',
  latvia: 'lv',
  lebanon: 'lb',
  lesotho: 'ls',
  liberia: 'lr',
  libya: 'ly',
  liechtenstein: 'li',
  lithuania: 'lt',
  luxembourg: 'lu',

  // M
  macau: 'mo',
  madagascar: 'mg',
  malawi: 'mw',
  malaysia: 'my',
  maldives: 'mv',
  mali: 'ml',
  malta: 'mt',
  mauritania: 'mr',
  mauritius: 'mu',
  mexico: 'mx',
  moldova: 'md',
  monaco: 'mc',
  mongolia: 'mn',
  montenegro: 'me',
  morocco: 'ma',
  mozambique: 'mz',
  myanmar: 'mm',

  // N
  namibia: 'na',
  nepal: 'np',
  netherlands: 'nl',
  'holland': 'nl',
  'new zealand': 'nz',
  nicaragua: 'ni',
  niger: 'ne',
  nigeria: 'ng',
  'north macedonia': 'mk',
  macedonia: 'mk',
  norway: 'no',
  'northern ireland': 'gb-nir',

  // O
  oman: 'om',

  // P
  pakistan: 'pk',
  palestine: 'ps',
  panama: 'pa',
  'papua new guinea': 'pg',
  paraguay: 'py',
  peru: 'pe',
  philippines: 'ph',
  poland: 'pl',
  portugal: 'pt',
  'puerto rico': 'pr',

  // Q
  qatar: 'qa',

  // R
  romania: 'ro',
  russia: 'ru',
  rwanda: 'rw',

  // S
  'saudi arabia': 'sa',
  scotland: 'gb-sct',
  senegal: 'sn',
  serbia: 'rs',
  seychelles: 'sc',
  'sierra leone': 'sl',
  singapore: 'sg',
  slovakia: 'sk',
  slovenia: 'si',
  somalia: 'so',
  'south africa': 'za',
  'south sudan': 'ss',
  spain: 'es',
  'sri lanka': 'lk',
  sudan: 'sd',
  suriname: 'sr',
  sweden: 'se',
  switzerland: 'ch',
  syria: 'sy',

  // T
  tajikistan: 'tj',
  tanzania: 'tz',
  thailand: 'th',
  'timor-leste': 'tl',
  togo: 'tg',
  'trinidad and tobago': 'tt',
  tunisia: 'tn',
  turkey: 'tr',
  turkmenistan: 'tm',

  // U
  uganda: 'ug',
  ukraine: 'ua',
  'united arab emirates': 'ae',
  uae: 'ae',
  'united states': 'us',
  usa: 'us',
  uruguay: 'uy',
  uzbekistan: 'uz',

  // V
  vanuatu: 'vu',
  venezuela: 've',
  vietnam: 'vn',
  'viet nam': 'vn',

  // W
  wales: 'gb-wls',

  // Y
  yemen: 'ye',

  // Z
  zambia: 'zm',
  zimbabwe: 'zw',
}

function normalizeTeamName(name: string): string {
  return name
    .normalize('NFD')
    .replace(/\p{M}/gu, '')
    .toLowerCase()
    .replace(/&/g, 'and')
    .replace(/['’.]/g, '')
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
}

/**
 * Resolve a participant label to a Flagpedia ISO code, or null for unknown clubs.
 */
export function teamNameToFlagCode(name: string): string | null {
  const trimmed = name.trim()
  if (!trimmed) return null

  const normalized = normalizeTeamName(trimmed)
  if (!normalized) return null

  const direct = TEAM_ALIASES[normalized]
  if (direct) return direct

  // "USA U21", "Spain Women" → strip suffix tokens
  const stripped = normalized
    .replace(/\b(u\d{1,2}|u21|u23|women|w|youth|olympic|olympics)\b/g, '')
    .replace(/\s+/g, ' ')
    .trim()

  if (stripped && stripped !== normalized) {
    return TEAM_ALIASES[stripped] ?? null
  }

  return null
}