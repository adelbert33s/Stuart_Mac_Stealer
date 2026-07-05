#!/usr/bin/env python3
"""Generate wallet_extensions.go from reference extension ID lists."""
import json
import re
import urllib.request

EXCLUDE = {
    "aeblfdkhhhdcdjpifhhbdiojplfjncoa",
    "khgocmkkpikpnmmkgmdnfckapcdkgfaf",
    "gejiddohjgogedgjnonbofjigllpkmbf",
    "hdokiejnpimakedhajhdlcegeplioahd",
    "ghmbeldphafepmbegfdlkpapadhbakde",
    "fdjamakpfbbddfjaooikfcpapjohcfmg",
    "bhghoamapcdpbohphigoooaddinpkbai",
    "pioclpoplcdbaefihamjohnefbikjilc",
}

NAMES = {
    "nkbihfbeogaeaoehlefnkodbefgpgknn": "MetaMask",
    "ejbalbakoplchlghecdalmeeeajnimhm": "MetaMask",
    "bfnaelmomeimhlpmgjnjophhpkkoljpa": "Phantom",
    "egjidjbpglichdcondbcbdnbeeppgdph": "Trust Wallet",
    "hnfanknocfeofbddgcijnmhnfnkdnaad": "Coinbase Wallet",
    "acmacodkjbdgmoleebolmdjonilkdbch": "Rabby",
    "mcohilncbfahbmgdjkbpemcciiolgcge": "OKX Wallet",
    "ppbibelpcjmhbdihakflkdcoccbgbkpo": "UniSat",
    "jnlgamecbpmbajjfhmmmlhejkemejdma": "Braavos",
    "aflkmfhebedbjioipglgcbcmnbpgliof": "Backpack",
    "ldinpeekobnhjjdofggfgjlcehhmanlj": "Leather",
    "onhogfjeacnfoofkfgppdlbmlmnplgbn": "SubWallet",
    "jiidiaalihmmhddjgbnbgdfflelocpak": "Bitget Wallet",
    "pdliaogehgdbhbnmkklieghmmjkpigpa": "Bybit Wallet",
    "hifafgmccdpekplomjjkcfgodnhcellj": "Crypto.com Wallet",
    "dlcobpjiigpikoobohmabehhmhfoodbb": "Argent X",
    "klghhnkeealcohjjanjjdaeeggmfmlpl": "Zerion",
    "dmkamcknogkgcdfhhbddcghachkejeap": "Keplr",
    "fnjhmkhhmkbjkkabndcnnogagogbneec": "Ronin",
    "aholpfdialjgjfhomihkjbmgjidlcdno": "Exodus Web3",
    "ibnejdfjmmkpcnlpebklmnkoeoihofec": "TronLink",
    "ffnbelfdoeiohenkjibnmadjiehjhajb": "Yoroi",
    "ookjlbkiijinhpmnjffcofjonbfbgaoc": "Temple Tezos",
    "bhhhlbepdkbapadjdnnojkbgioiodbic": "Solflare",
    "lgmpcpglpngdoalbgeoldeajfclnhafa": "SafePal",
    "mfgccjchihfkkindfppnaooecgfneiii": "TokenPocket",
    "nphplpgoakhhjchkkhmiggakijnkhfnd": "Ton Wallet",
    "idnnbdplmphpflfnlkomgpfbpcgelopg": "Xverse",
    "kkpllkodjeloidieedojogacfhpaihoh": "Enkrypt",
    "cphhlgmgameodnhkjdmkpanlelnlohao": "NeoLine",
    "nhnkbkgjikgcigadomkphalanndcapjk": "CLV Wallet",
    "mkpegjkblkkefacfnmkajcjmabijhclg": "Magic Eden",
    "fcfcfllfndlomdhbehjjcoimbgofdncg": "Leap Cosmos",
    "aijcbedoijmgnlmjeegjaglmepbmpkpi": "Leap Terra",
    "khpkpbbcccdmmclmpigdgddabeilkdpd": "Suiet",
    "loinekcabhlmhjjbocijdoimmejangoa": "Glass Wallet Sui",
    "ocjdpmoallmgmjbbogfiiaofphbjgchh": "Elli Sui",
    "ehgjhhccekdedpbkifaojjaefeohnoea": "Ambire",
    "eaeecbmeajhliilmacefcgjnnijkkfki": "Trust Wallet Beta",
    "fhbohimaelbohpjbbldcngcnapndodjp": "Binance Chain Wallet",
    "fihkakfobkmkjojpchpfgcmhfjnmnfpi": "BitApp Wallet",
    "aodkkagnadcbobfpggfnjeongemjbjca": "BoltX",
    "aeachknmefphepccionboohckonoeemg": "Coin98",
    "agoakfejjabomempkjlepdflaleeobhb": "Core Wallet",
    "pnlfjmlcjdjgkddecgincndfgegkecke": "Crocobit",
    "blnieiiffboillknjnepogjhkgnoapac": "Equal Wallet",
    "cgeeodpfagjceefieflmdfphplkenlfk": "Ever Wallet",
    "ebfidpplhabeedpnhjnobghokpiioolj": "Fewcha",
    "cjmkndjhnagcfbpiemnkdpomccnjblmj": "Finnie",
    "hpglfhgfnhbgpjdenjgmdgoeiappafln": "Guarda",
    "nanjmdknhkinifnkgdcggcfnhdaammmj": "Guild Wallet",
    "fnnegphlobjdpkhecapkijjdkgcjhkib": "Harmony Wallet",
    "flpiciilemghbmfalicajoolhkkenfel": "Iconex",
    "cjelfplplebdjjenllpjcblmjkfcffne": "Jaxx Liberty",
    "jblndlipeogpafnldhgmapagcccfchpi": "Kaikas",
    "pdadjkfkgcafgbceimcpbkalnfnepbnk": "KardiaChain",
    "kpfopkelmapcoipemfendmdcghnegimn": "Liquality",
    "nlbmnnijcnlegkjjpcfjclmcfggfefdm": "MEW CX",
    "dngmlblcodfobpdpecaadgfbcggfjfnm": "Maiar DeFi",
    "efbglgofoippbgcjepnhiblaibcnclgk": "Martian",
    "afbcbjpbpfadlkmhmclhkeeodmamcflc": "Math Wallet",
    "fcckkdbjnoikooededlapcalpionmalo": "Mobox",
    "lpfcbjknijpeeillifnkikgncikgfhdo": "Nami",
    "jbdaocneiiinmjbjlgalhcelgbejmnid": "Nifty Wallet",
    "fhilaheimglignddkjgofkcbgekhenbh": "Oxygen Wallet",
    "mgffkfbidihjpoaomajlbgchddlicgpn": "Pali Wallet",
    "ejjladinnckdgjemekebdpeokbikhfci": "Petra",
    "phkbamefinggmakgklpkljjmgibohnba": "Pontem",
    "nkddgncdjgjfcddamfgcmfnlhccnimig": "Saturn Wallet",
    "pocmplpaccanhmnllbbkpgfliimjljgo": "Slope",
    "fhmfendgdocmcbmfikdcogofphimnkno": "Sollet",
    "mfhbebgoclkghebffdldpobeajmbecfk": "Starcoin",
    "cmndjbecilbocjfkibfbifhngkdmjgog": "Swash",
    "aiifbnbfobpmeekipheeijimdpnlpgpp": "Terra Station",
    "amkmjjmmflddogmhpjloimipbofnfjih": "Wombat",
    "hmeobnfnfcmdkdcmlblgagmfpfboieaf": "XDEFI",
    "eigblbgjknlfbajkfhopmcojidlgcehm": "XMR.PT",
    "bocpokimicclpaiekenaeelehdjllofo": "XinPay",
    "kncchdigobghenbbaddojjnnaogfppfj": "iWallet",
    "opcgpfmipidbgpenhmajoajpbobppdil": "Sui Wallet",
    "oebgglckkdmdcphmbdcbdlkedjbbinii": "Sender",
    "hifafgmccdpekplomjjkcfgodnhcellj": "Crypto.com Wallet",
}


def main():
    url = "https://raw.githubusercontent.com/Darksp33d/hyperhives-macos-infostealer-analysis/main/output/full_decrypted_config.json"
    data = json.loads(urllib.request.urlopen(url, timeout=30).read())
    ids = sorted(set(data.get("extension_ids", [])))
    name_by_id = dict(NAMES)

    for val in data.get("all_strings", {}).values():
        m = re.search(r"([a-z]{32})$", val.lower())
        if not m:
            continue
        eid = m.group(1)
        prefix = val[: len(val) - 32].strip()
        if eid in name_by_id or not prefix or len(prefix) < 3:
            continue
        cleaned = re.sub(r"[^A-Za-z0-9 |&.-]", "", prefix).strip()
        if cleaned:
            name_by_id[eid] = cleaned[:56]

    final = {}
    for eid in ids:
        if eid in EXCLUDE:
            continue
        name = name_by_id.get(eid, "Wallet")
        final[eid] = name

    out_path = "recovery/scanner/wallet_extensions.go"
    lines = [
        "package scanner",
        "",
        "// knownWalletExtensions maps Chromium extension IDs to crypto wallet names.",
        "// IDs are matched for extension metadata and Local Extension Settings harvesting.",
        "var knownWalletExtensions = map[string]string{",
    ]
    for eid in sorted(final.keys()):
        name = final[eid].replace('"', '\\"')
        lines.append('\t"{}": "{}",'.format(eid, name))
    lines.append("}")
    lines.append("")

    with open(out_path, "w") as f:
        f.write("\n".join(lines))

    print("Wrote {} wallet IDs to {}".format(len(final), out_path))


if __name__ == "__main__":
    main()