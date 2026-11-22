---
name: "hongaarse-woordenlijst-csv"
description: "Zet een Hongaarse woordenlijst (PDF of geplakte aantekeningen) om naar een importbestand voor de leerapp: hongaars;nederlands;context;voorbeeldzin;woordsoort;vormen, alleen basisvormen, met woordsoort en de uitzonderingen op de grammaticaregels van de app."
---

# Hongaarse woordenlijst naar importbestand

Gebruik dit wanneer de gebruiker een Hongaarse woordenlijst, glossary-PDF of lesaantekeningen aanlevert die omgezet moeten worden naar een bestand voor de leerapp. Communiceer in het Nederlands.

De app leert woorden alleen in hun basisvorm en maakt de andere vormen (naamvallen, meervoud, vervoegingen) zelf met regels, in het tabblad Regels. Dit bestand levert daarom: de basisvormen, hun woordsoort, en per woord de vormen waar die regels de mist in zouden gaan.

Deze skill volgt de regels in de app (`internal/grammar`, stand 2026-11-21, versie 5: -lak/-lek, gebiedende wijs). Komt er logica bij in de app, dan wordt deze skill bijgewerkt.

## Bestand
- CSV, scheidingsteken puntkomma `;`, UTF-8.
- Kopregel: `hongaars;nederlands;context;voorbeeldzin;woordsoort;vormen`
- Elke regel heeft precies 6 kolommen (lege kolom 6 mag: `ház;huis;hier woon je;Ez egy ház.;zn;`).
- Geen `;` binnen een kolom.
- Lever het bestand op met SendUserFile.

## Kolom 1: hongaars
- Precies één los woord: geen spaties en geen leestekens (geen ? ! / ( ) - . , = ~).
- Alles in kleine letters, ook eigennamen.
- Varianten met een slash splitsen: `honnan/honnét` geeft twee regels.
- Aanvullingen zoals vmit, vhol, vhova, vmire, van weglaten (`eszik vmit` -> eszik, `nyitva van` -> nyitva). Tekst tussen haakjes of na `=` weglaten.
- Woorden die alleen met een koppelteken bestaan (bijv. négyezer-ötszázkilencven) overslaan en dat melden.

## Alleen het basiswoord, geen vormen die de app zelf maakt
- Neem geen vormen op met een naamvalsuitgang: -ban/-ben (városban), -ba/-be, -ból/-ből, -n/-on/-en/-ön (hétfőn, kedden), -ra/-re, -ról/-ről, -nál/-nél, -hoz/-hez/-höz, -tól/-től, -nak/-nek, -val/-vel, -vá/-vé, -ig, -ért, -ként, -kor (éjfélkor, ötkor), -szor/-szer/-ször (ötször, egyszer, kétszer).
- Geen naamvallen met een persoon: nekem, nálam, nála, tőle, velem, vele, hozzá, benne, belőle, rajta, róla, rá, érte (enz.). Die maakt de app met de regel "Naamval met persoon".
- Geen trappen van vergelijking: kisebb, legkisebb, jobb (beter), nagyobb, legszebb, később. Neem alleen het bijvoeglijk naamwoord zelf op (kicsi, jó, nagy, szép, késő).
- Uitzondering op deze drie: woorden met een eigen betekenis blijven (sokszor = vaak, először = eerst, legkésőbb = uiterlijk, hánykor = hoe laat). Twijfel je, meld het.
- Geen lijdend voorwerp (-t/-ot/-et/-öt: szót) en geen meervoud (-k/-ok/-ek/-ök).
- Geen bezittelijke vormen: -m/-om/-em/-am (barátom), -d/-od/-ed (füled), -a/-e/-ja/-je (háza, neve), meervoudige bezitters enz.
- Zet zo'n vorm om naar het woord waar het over gaat (városban -> város, fül/ed -> fül, barátom -> barát, éjfélkor -> éjfél) en neem alleen dat op. Een slash die een uitgang afscheidt (`szomszéd/od`) betekent alleen het basiswoord (szomszéd).
- Uitzondering: woorden die als vaste uitdrukking een eigen betekenis hebben blijven (általában, rendben, balra, jobbra, magyarul). Twijfel je, meld het.
- Zit er een kleinere stam of los woord in een samenstelling of afleiding, neem dat dan eventueel ook op als eigen regel (bevásárlóközpont -> ook központ; szerencsére -> szerencse; nemzetiségű -> nemzetiség), als dat nuttig is om te leren.
- Geen bijvoeglijke naamwoorden op -i van een plaats of tijd (budapesti, londoni, utrechti, győri, reggeli, nyári, heti): die maakt de app met de -i-regel, uit een eigen lijst met plaatsen en tijdwoorden. Plaatsnamen zelf hoef je ook niet op te nemen, tenzij ze als woord in de les staan (magyarország).
- Echte afleidingen die een nieuw woord met een eigen betekenis vormen zijn wel aparte woorden: bijvoeglijke naamwoorden op -i met een eigen betekenis (közlekedési, emberi, nemzetközi, külföldi), op -s, -ú/-ű, bijwoorden op -an/-en (ritkán, nagyon), samenstellingen.

## Werkwoorden
- Neem van elk werkwoord alleen de infinitief en de 3e persoon enkelvoud op, elk als eigen regel. De app maakt de andere personen zelf uit de 3e persoon.
  - Infinitief -> Nederlandse infinitief (főzni = koken).
  - 3e persoon enkelvoud -> Nederlandse 3e persoon enkelvoud zonder voornaamwoord (alszik = slaapt, megy = gaat).
- Andere vervoegde vormen (mondok, beszélsz, mosolygunk, olvasott, mentem, olvashat, fogok, szeretlek, olvass, menj, gyere, legyen, ...) niet opnemen; zet ze om naar het werkwoord en neem dan de infinitief en 3e persoon enkelvoud op. De app maakt de tegenwoordige en verleden tijd, de toekomende tijd met fog, -hat/-het, -lak/-lek en de gebiedende wijs zelf. Alleen *lehet* (kan, het is mogelijk) blijft als eigen woord.
- Wederkerend 'zich' vervalt als dat nodig is (fejlődik = ontwikkelt).
- Neem altijd **allebei** op, ook als het bronmateriaal er maar één geeft: de app vervoegt vanuit de 3e persoon, en de import waarschuwt als een werkwoord zonder partner binnenkomt. Let op -ik-werkwoorden: dolgozni → dolgozik, esni → esik, reggelizni → reggelizik.
- Context altijd in de vaste vorm (zie kolom 3), woordsoort `ww` bij allebei. De app zet werkwoorden zelf in de groep Werkwoorden.
- Heeft het Nederlandse woord al een andere betekenis in de lijst (vraagt = kérdez en kér, koopt = vásárol en vesz), geef dan beide een betekenis in de context: `werkwoord · 3e persoon enkelvoud · om iets vragen`.

## Kolom 2: nederlands
- Precies één vertaling per regel. Meerdere betekenissen (gescheiden door , of /) -> aparte regels met hetzelfde Hongaarse woord, elk met een eigen context die de betekenis aanwijst.
- Kleine letters, geen leestekens. Meerdere woorden mag als er geen los Nederlands woord voor bestaat (naar huis, blij zijn, ver weg).
- Klopt de vertaling in het bronmateriaal niet (bijv. ritkán = zelden, niet 'af en toe'; lány = meisje, niet 'dochter'), gebruik dan de juiste en meld dat.
- Maak alleen één regel met twee Nederlandse woorden als ze echt hetzelfde betekenen (szia -> hoi en hallo). Maakt het Nederlands een onderscheid dat het Hongaars niet maakt (ő -> hij / zij), dan twee regels met elk een context.

## Kolom 3: context
- Korte context in het Nederlands die het antwoord niet verraadt (bijv. ágy -> 'hier slaap je in').
- Bij varianten en dubbele betekenissen aangeven welke bedoeld is ('variant van hova', 'het gebouw van de agenten').
- **Werkwoorden in vaste vorm:** `werkwoord · infinitief` of `werkwoord · 3e persoon enkelvoud`, eventueel met de betekenis erachter: `werkwoord · infinitief · vuur maken`. Gebruik het teken `·` (middenpunt) met spaties eromheen.

## Kolom 4: voorbeeldzin
- Eenvoudige, grammaticaal correcte Hongaarse zin met normale hoofdletters en leestekens.
- Bevat het Hongaarse woord exact in de vorm van kolom 1 (als los woord), dus in de basisvorm zonder uitgang.
- Gebruik de voorbeeldzin uit het lesmateriaal als die het exacte woord bevat en correct is; anders zelf een zin schrijven.

## Kolom 5: woordsoort
De woordsoort van het **Hongaarse** woord, als code:

| code | woordsoort | voorbeeld |
|---|---|---|
| zn | zelfstandig naamwoord (ook eigennamen) | ház, magyarország |
| ww | werkwoord (infinitief én 3e persoon) | olvas, olvasni |
| bn | bijvoeglijk naamwoord | szép, budapesti |
| bw | bijwoord (ook tijd/plaats: ma, itt, hétfőn) | gyakran, haza |
| vnw | voornaamwoord (ook vragend: mi, ki, melyik, mennyi) | én, ez, valami |
| telw | telwoord (ook fél, negyed, néhány) | három, két |
| lw | lidwoord | a, az |
| vw | voegwoord | és, de, mert, hogy |
| vz | achterzetsel (névutó) | ellen, után, mellett |
| tw | tussenwerpsel, groet, ja/nee | szia, igen, bocsánat |
| uitdr | uitdrukking van meer woorden | jó reggelt |

De regels in de app werken op `zn` (naamvallen, -t, -k), `ww` (vervoegen, infinitief), `bn` (vergrotende en overtreffende trap) en `telw` (-szor/-szer/-ször). De woordsoort bepaalt dus welke regels bij het woord horen: kies hem zorgvuldig. (-i, -kor en de naamval met persoon oefent de app met eigen vaste lijsten.)

## Kolom 6: vormen (uitzonderingen)
Leeg, tenzij de regels van de app (hieronder) voor dit woord een verkeerde vorm zouden maken, of het woord anders niet geoefend wordt. Schrijfwijze: `sleutel=vorm`, gescheiden door komma's: `t=házat, k=házak`. Een `?` als vorm betekent "niet zeker, niet oefenen".

**Sleutels**
- Zelfstandige naamwoorden: `t` (lijdend voorwerp), `k` (meervoud), `ban`, `ba`, `ból`, `on` (de -n/-on/-en/-ön-vorm), `ra`, `ról`, `nál`, `hoz`, `tól`, `nak`, `val`, `vá`, `ig`, `ért`, `ként`, `i` (bijvoeglijk op -i), `kor`.
- Bijvoeglijke naamwoorden: `bb` (vergrotende trap), `leg` (overtreffende trap).
- Telwoorden: `szor` (… keer).
- Werkwoorden (uit de 3e persoon): `inf` (infinitief); onbepaald `én`, `te`, `ő`, `mi`, `ti`, `ők`; bepaald `bep-én`, `bep-te`, `bep-ő`, `bep-mi`, `bep-ti`, `bep-ők`; verleden tijd `mult-én` … `mult-ők` en bepaald `multbep-én` … `multbep-ők`; `hat` (de -hat/-het-vorm, bijv. `hat=lehet`); `lak` (ik … jou); gebiedende wijs `felsz-én` … `felsz-ők` en bepaald `felszbep-én` … `felszbep-ők`; en `bep=?` voor een werkwoord zonder lijdend voorwerp (geen bepaalde vervoeging, geen bepaalde gebiedende wijs en geen -lak, bijv. sétál, marad). De toekomende tijd volgt uit `inf`.

**Wat de app zelf al goed doet (dan kolom 6 leeg laten)**

*Klinkerharmonie.* Na de laatste achterklinker (a á o ó u ú) alleen nog i/í/é of niets: achter (-ban, -hoz). Anders voor (-ben, -hez); bij drie varianten rond (-höz, -öt) als de laatste klinker ö ő ü ű is. Samenstellingen op gép, kép, szék, rész, nép, hír, év zijn voor (számítógépben). Achterklinker + e (hotel, fotel) of + twee keer i/í/é (ábécé): beide varianten goed. Uitzonderingen die de app kent: híd, íj, nyíl, zsír (achter), en de werkwoorden ír, nyit, szív, hív, sír, nyír, iszik (achter).

*Naamvallen.* Basiswoord + uitgang; eind-a/-e wordt á/é (alma -> almában). -n na een klinker, anders -on/-en/-ön. -val/-vel en -vá/-vé: na een medeklinker wordt de v gelijk aan die medeklinker (busz -> busszal, kanál -> kanállal, toll -> tollal). -i zonder verlenging, kleine letter (Amerika -> amerikai); die regel oefent alleen met de vaste lijst plaatsen en tijdwoorden in de app (Budapest, Győr, Bécs, London, reggel, este -> esti, hét -> heti, hónap -> havi …), dus de sleutel `i` heb je bijna nooit nodig.

*-t en -k.* Na een klinker direct (met á/é). -t direct na j, l, ly, n, ny, r, s, sz, z, zs; anders -ot/-et/-öt. Meervoud altijd met verbindingsklinker -ok/-ek/-ök. **Maar:** eenlettergrepige woorden en woorden die eindigen op korte klinker + losse medeklinker (sarok, asztal, bokor) oefent de app voor -t en -k **niet**, tenzij ze in de lijsten staan. Vul bij zulke woorden dus altijd `t=` en `k=` in, ook als ze regelmatig zijn (`kar;arm;lichaamsdeel;Eltört a kar.;zn;t=kart, k=karok`), anders worden ze niet geoefend.

Woorden die de app al kent als uitzondering (geen kolom 6 nodig): haj, lyuk, vonal, szakáll, társ, tár, hely, hét, tér, úr, kút, titok, fakanál, étterem, könyv, föld, fül, ház, fal, hal, toll, vaj, fej, száj, ágy, fog, hold, vár, gyár, nyár, út, híd, madár, kanál, pohár, szamár, kenyér, levél, egér, tél, név, kéz, víz, tűz, ló, tó, hó, kő, fű, mű, szó, sarok, bokor, cukor, szobor, tükör, álom, majom, fogalom, dolog, terem, eper, torony, selyem, lélek, teher, falu. Samenstellingen volgen hun laatste deel (kórház als ház, munkahely als hely) als het eerste deel minstens drie letters heeft; bij een korter eerste deel (fa-, fő-) kolom 6 invullen.

Als regelmatig bekend (ook als laatste deel van een samenstelling): alak, arc, asztal, bank, város, beteg, bolt, busz, citrom, csend, csütörtök, cím, dokumentum, dzsip, dzsungel, szünet, eszköz, fejezet, feladat, felület, film, fjord, orvos, fotel, másolat, fókusz, főnök, gyakorlat, gyalogos, gyerek, gép, hang, hegy, helyzet, nap, hónap, hőmérséklet, internet, jog, kedd, kozmetikus, kukac, kép, laptop, lány, jegy, telefon, mondat, motor, füzet, mérséklet, szak, nyelv, nyelvtan, nyugat, park, perc, program, udvar, péntek, rag, reggel, sport, szombat, szám, szín, sör, templom, tenger, természet, tyúk, típus, tök, vicc, villamos, vonat, xilofon, áram, szer, épület, év, üveg, üzlet, kert, kés, pénz, szék, ablak, húg, amszterdam, főváros, rész, barát, sor, kör, bor, tej.

*-kor en -szor.* -kor: woord + kor, geen harmonie, geen verlenging (ötkor, éjfélkor, vacsorakor); de app oefent het met een vaste lijst klokuren en feestdagen. -szor/-szer/-ször: telwoord + uitgang met klinkerharmonie, zonder verlenging (háromszor, egyszer, ötször); kettő → kétszer. Je eigen telwoorden doen mee (huszonötször).

*Vergelijken (bn).* Na een klinker -bb, eind-a/-e wordt á/é (drága → drágább, fekete → feketébb, olcsó → olcsóbb, késő → később). Na een medeklinker -abb (achter) of -ebb (voor en rond): magas → magasabb, kedves → kedvesebb, zöld → zöldebb. Overtreffend: leg- + vergrotend (legdrágább). De app kent als uitzondering: jó → jobb, sok → több, kicsi/kis → kisebb, szép → szebb, nagy → nagyobb, öreg → idősebb, kevés → kevesebb, nehéz → nehezebb, bő → bővebb, régi → régebbi, hosszú → hosszabb, lassú → lassabb, könnyű → könnyebb, ifjú → ifjabb, bátor → bátrabb, keserű → keserűbb, szomorú → szomorúbb, savanyú → savanyúbb, gyönyörű → gyönyörűbb, egyszerű → egyszerűbb. Niet vergeleken (geen kolom 6 nodig): bal, jobb, nationaliteiten (magyar, holland …), egyéb, külön, fél, első, utolsó, kedvenc, normális. Bijvoeglijke naamwoorden op -i, -ú of -ű die niet in de lijst staan, oefent de app niet.

*Naamval met persoon.* Vaste tabel in de app (nekem, nálam, tőlem, velem, hozzám, bennem, belőlem, belém, rajtam, rólam, rám, értem en de andere personen); daar hoeft niets voor in het bestand.

*Werkwoorden, onbepaald.* De stam is de 3e persoon zonder -ik (lakik -> lak). "Twee medeklinkers" betekent: de stam eindigt op twee verschillende medeklinkerklanken, zoals -nd (mond), -ld (küld), -rt (tart, ért), -st (fest), -jt (gyújt), -tsz (játszik). Een klank van twee letters (sz, gy, ny, zs, cs, ly, ty) telt als één medeklinker; een dubbele (áll, hall) telt niet als twee.

| persoon | gewoon | na s, sz, z | na -ít of twee medeklinkers |
|---|---|---|---|
| én | -ok/-ek/-ök (-ik-ww: -om/-em/-öm) | zelfde | zelfde (mondok, értek) |
| te | -sz (tanulsz) | -ol/-el/-öl (olvasol, nézel, főzöl) | -asz/-esz (mondasz, küldesz, tanítasz, értesz) |
| ő | de stam (+ -ik) | | |
| mi | -unk/-ünk | zelfde | zelfde |
| ti | -tok/-tek/-tök | zelfde (olvastok) | **alleen** na -ít of na medeklinker + t/s/sz/z: -otok/-etek/-ötök (tanítotok, tartotok, értetek, festetek, játszotok); na -nd, -ld, -zd enz. gewoon: mondtok, küldtök, kezdtek |
| ők | -nak/-nek | zelfde (olvasnak) | -anak/-enek (mondanak, küldenek, tartanak, játszanak) |

Klinkerharmonie bepaalt de variant (achter -ok/-unk/-nak, voor -ek/-ünk/-nek, rond -ök/-tök); bij -asz/-esz, -unk/-ünk, -nak/-nek en -anak/-enek telt rond als voor. Infinitief: stam + -ni (olvasni, lakni); na -ít of twee medeklinkers -ani/-eni (mondani, küldeni, tartani, érteni, tanítani, játszani); -ni zelf doet niet mee aan harmonie.

Voorbeelden zoals de app ze maakt:
- mond: mondok, mondasz, mond, mondunk, mondtok, mondanak; mondani
- küld: küldök, küldesz, küld, küldünk, küldtök, küldenek; küldeni
- tart: tartok, tartasz, tart, tartunk, tartotok, tartanak; tartani
- ért: értek, értesz, ért, értünk, értetek, értenek; érteni
- segít: segítek, segítesz, segít, segítünk, segítetek, segítenek; segíteni
- játszik: játszom, játszol, játszik, játszunk, játszotok, játszanak; játszani
- olvas: olvasok, olvasol, olvas, olvasunk, olvastok, olvasnak; olvasni

*Werkwoorden, bepaald:* -om/-od/-ja/-juk/-játok/-ják (achter), -em/-ed/-i/-jük/-itek/-ik (voor; rond -öm/-öd), ook na twee medeklinkers zonder extra klinker (mondom, mondod, mondja, mondjuk, mondjátok, mondják; értem, érted, érti, értjük, értitek, értik). Na s/sz/z wordt de j gelijk aan die medeklinker (olvassa, olvassuk, nézzük, játssza). -ik-werkwoorden hebben geen bepaalde vervoeging, behalve eszik, iszik, játszik.

*Verleden tijd.* Verleden-stuk = stam + -t- (tanult-, olvast-, mondt-), met -ott/-ett/-ött in alle personen na -ít, na medeklinker + t/s/sz/z (ért, tart, fest, játsz) en bij stammen van één lettergreep op -t (fut → futottam). Uitgangen onbepaald: -am/-em, -ál/-él, -unk/-ünk, -atok/-etek, -ak/-ek; bepaald: -am/-em, -ad/-ed, -a/-e, -uk/-ük, -átok/-étek, -ák/-ék. Voor ő: na één j, l, ly, n, ny, r alleen -t (tanult, beszélt, várt, írt); anders -ott/-ett/-ött (olvasott, nézett, főzött, mondott, dolgozott, tudott). Klinkerharmonie van het werkwoord zelf (ír → írtam). Onregelmatige verleden tijden die de app kent: van (volt), lesz (lett), megy (ment), jön (jött), eszik (ettem, evett), iszik (ittam, ivott), alszik (aludt), fekszik (feküdt), haragszik (haragudott), tesz (tett), vesz (vett), visz (vitt), hisz (hitt), lő (lőtt), nő (nőtt), fürdik (fürödtem, fürdött), hall (hallott), áll (állt), száll (szállt), lát (láttam, látott), marad (maradt), mosolyog (mosolygott). De app slaat over: meerlettergrepige werkwoorden op -d zonder -ik (enged), op een dubbele medeklinker of op een klinker; vul daar `mult-…` in.

*Toekomende tijd en -hat/-het.* fog + infinitief (olvasni fogok / fogok olvasni), dus alleen een afwijkende `inf` doet ertoe. -hat (achter) / -het (voor) achter de stam, daarna als gewoon werkwoord vervoegd (olvashatok, nézhettek). Onregelmatig bekend: van/lesz → lehet, megy → mehet, jön → jöhet, eszik → ehet, iszik → ihat, tesz → tehet, vesz → vehet, visz → vihet, hisz → hihet, alszik → alhat, fekszik → fekhet. Een ander werkwoord met een onregelmatige stam: `hat=…`.

*-lak/-lek (ik … jou).* Stam + -lak (achter) / -lek (voor): látlak, szeretlek, várlak; na -ít of twee medeklinkers -alak/-elek (tanítalak, értelek, mondalak); stam op -ll zonder derde l (hallak). Alleen bij werkwoorden met een lijdend voorwerp.

*Gebiedende wijs.* Er komt een j tussen stam en uitgang: -jak/-jek, -j (lang -jál/-jél), -jon/-jen/-jön, -junk/-jünk, -jatok/-jetek, -janak/-jenek; bepaald -jam/-jem, -d (lang -jad/-jed), -ja/-je, -juk/-jük, -játok/-jétek, -ják/-jék. De j smelt samen: na s/sz/z wordt hij die medeklinker (olvass, nézz, játssz, ússz), klinker + t wordt ss (láss, fizess, fuss), -ít en medeklinker + t worden ts (taníts, érts, tarts), st wordt ss (fess), szt wordt ssz. Korte bepaalde jij-vorm: olvasd, nézd, tanuld, lásd, tanítsd, fesd, mondd. Onregelmatig bekend: van/lesz (legyen), megy (menj), jön (gyere, gyertek, jöjjön), eszik (egyél, edd), iszik (igyál, idd), tesz (tegyél, tedd), vesz (vegyél, vedd), visz (vigyél, vidd), hisz (higgyél, hidd), alszik (aludj), fekszik (feküdj), fürdik (fürödj). Werkwoorden op een klinker (lő) en op een dubbele medeklinker behalve ll slaat de app over; vul daar `felsz-…` in.

*Voorvoegsels.* Een werkwoord met voorvoegsel volgt het werkwoord zelf, ook in de klinkerharmonie: felhív als hív (felhívok, felhívlak), megír als ír.

Onregelmatige werkwoorden die de app kent (ook met voorvoegsel, elmegy = el + megy): van, lesz, megy, jön, eszik, iszik, alszik, fekszik, haragszik, tesz, vesz, visz, hisz, fürdik, hall, áll, száll. Zonder bepaalde vervoeging bekend: van, lesz, megy, jön, él, marad, sétál, táncol, fordul, jár, nyaral, pihen, ebédel, vacsorázik, reggelizik, koncentrál, dönt, tankol, áll, száll, fut, ül, siet, mosolyog, örül, repül, kiabál, telefonál, sportol, sír, nevet, fél, ér. Niet vervoegd: nincs, lehet, kell, szabad, tilos, szokott.

**Wanneer kolom 6 invullen**
- Zelfstandig naamwoord dat zijn stam verandert en niet in de lijst staat: verlaagde verbindingsklinker (`t=falat`), korter wordende klinker (`t=szekeret, k=szekerek`), v-stam (`t=lovat, k=lovak, on=lovon`), wegvallende klinker (`t=bokrot, k=bokrok, on=bokron`).
- Eenlettergrepig of korte klinker + losse medeklinker en niet in de lijsten: altijd `t=` en `k=`.
- Afwijkende klinkerharmonie (een i/í-woord met achter-uitgangen): de vormen die je wilt laten oefenen, bijv. `ban=…, t=…, k=…`.
- Werkwoord met onregelmatige vormen die niet in de lijst staan: de afwijkende personen en/of `inf=`, `mult-…=`, `hat=`.
- Werkwoord waarvan de verleden tijd afwijkt (een klinker valt weg zoals mosolyog → mosolygott, of meerlettergrepig op -d zoals enged → engedett): `mult-ő=…` en zo nodig de andere `mult-…`.
- Werkwoord met stam op klinker + t (lát, süt, köt): de app slaat 'ti' over; vul `ti=` in (láttok, sütötök).
- Werkwoord met dubbele medeklinker aan het eind (hall, áll) of op medeklinker + sz + ik (alszik, fekszik, dicsekszik) dat niet in de lijst staat: `te=`, `ti=`, `ők=` en `inf=`.
- Bijvoeglijk naamwoord met een onregelmatige trap die niet in de lijst staat, of op -i, -ú, -ű: `bb=…, leg=…` (bijv. `bb=keserűbb, leg=legkeserűbb`), of `bb=?, leg=?` als het niet te vergelijken is (bijv. een nationaliteit of kleur-afleiding die de app niet kent).
- Werkwoord zonder lijdend voorwerp dat niet in de lijst staat (bijv. siet, ül zijn al bekend; utazik is een -ik-werkwoord en dus al goed): `bep=?`.
- Weet je een vorm niet zeker: `sleutel=?` (dan oefent de app die vorm niet) en meld het.

Vul alleen vormen in die je zeker weet. Kolom 6 bij een regel met de infinitief blijft leeg: alles staat bij de 3e persoon.

## Overslaan
- Losse uitgangen en grammaticaschema's (-szor/-szer/-ször, -a/-e/-ja/-je).
- Verwijzingen naar boek of werkboek (Tk., Mf., paginanummers).
- Hele zinnen en uitdrukkingen: opsplitsen in losse woorden (in basisvorm) in plaats van als geheel opnemen.
- Dubbele regels (zelfde Hongaars en zelfde Nederlands) maar één keer opnemen.

## Controle voor oplevering
Controleer met een script elke regel:
- precies 6 kolommen; kopregel klopt;
- kolom 1 en 2 in kleine letters; kolom 1 zonder spaties of leestekens; kolom 2 zonder leestekens;
- context gevuld; bij `ww` begint de context met `werkwoord · infinitief` of `werkwoord · 3e persoon enkelvoud`; infinitieven eindigen op -ni;
- voorbeeldzin bevat het exacte woord uit kolom 1 (woordgrens, hoofdletterongevoelig);
- kolom 5 is een van: zn, ww, bn, bw, vnw, telw, lw, vw, vz, tw, uitdr;
- kolom 6 is leeg of `sleutel=vorm`-paren gescheiden door komma's, met alleen de sleutels hierboven, één woord of `?` per vorm, `bep` alleen als `bep=?`; alleen bij zn, bn, telw en ww (3e persoon);
- elk werkwoord heeft zowel een infinitief- als een 3e-persoonregel;
- geen dubbele regels.
Controleer daarnaast handmatig dat er geen verbogen of bezittelijke vormen of extra werkwoordsvormen in kolom 1 staan. Los fouten op voor het opleveren.

## Afronding
Meld kort:
- het aantal regels, per woordsoort;
- de woorden met kolom 6 ingevuld, en waarom (bijv. "szekér: é wordt kort");
- uitzonderingen die je tegenkwam en die in de app thuishoren in de vaste lijsten (zodat ze daar kunnen worden toegevoegd);
- werkwoorden waarvan je de partner zelf hebt toegevoegd (de infinitief of de 3e persoon stond niet in het bronmateriaal);
- keuzes bij twijfelgevallen, wat is overgeslagen, en fouten in het bronmateriaal.
