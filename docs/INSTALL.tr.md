# Nginxlings kurulumu

Bu rehber, Nginxlings'i bir Linux bilgisayara kurmak ve açmak için gereken her
şeyi anlatır. Komut satırı bilgisi gerekmez; gereken birkaç komut olduğu gibi
kopyalanabilir.

## 1. Dosyayı indir

Eğitmeninin verdiği bağlantıdan ya da projenin GitHub sayfasındaki son sürümden
(release) kendi dağıtımına uygun dosyayı indir:

| Dağıtımın | İndireceğin dosya |
| --- | --- |
| Ubuntu, Debian, Linux Mint, Pop!_OS | `nginxlings_<sürüm>_amd64.deb` |
| Fedora, Rocky, AlmaLinux, RHEL, openSUSE | `nginxlings-<sürüm>-1.x86_64.rpm` |
| Arch, Manjaro | `nginxlings-<sürüm>-1-x86_64.pkg.tar.zst` |
| Diğerleri (Alpine, Void ...) | `nginxlings_<sürüm>_linux_amd64.tar.gz` |

Hangi dağıtımı kullandığını bilmiyorsan, bir terminal aç ve `cat /etc/os-release`
yaz; `NAME=` satırı dağıtımın adıdır. Bilgisayarın ARM işlemciliyse (ör. bazı
dizüstüler), adında `arm64` ya da `aarch64` geçen dosyayı indir.

## 2. Kur

İndirdiğin dosyaya çift tıkla. Uygulama merkezi (software center) açılırsa
"Kur" düğmesine bas ve şifreni gir.

Ubuntu'da çift tıklama paketi kurmaz, arşiv yöneticisini açar ("Çıkar" ya da
"Extract" düğmesi görünür). O pencereyi kapat, dosyaya sağ tıkla, "Birlikte Aç"
(Open With) menüsünden "Yazılım Kur"u (Software Install) seç, sonra "Kur"
düğmesine bas ve şifreni gir.

Bu da işe yaramazsa bir terminal aç, dosyayı indirdiğin klasöre geç ve
dağıtımına uyan komutu yaz:

```sh
cd ~/Downloads                                                # Türkçe kurulumda: cd ~/İndirilenler
sudo apt install ./nginxlings_*_amd64.deb                     # Ubuntu, Debian, Mint
sudo dnf install ./nginxlings-*.x86_64.rpm                    # Fedora, Rocky, Alma
sudo zypper install --allow-unsigned-rpm ./nginxlings-*.rpm   # openSUSE
sudo pacman -U ./nginxlings-*.pkg.tar.zst                     # Arch, Manjaro
```

`apt` "Unsupported file" derse terminal dosyanın bulunduğu klasörde değildir;
önce `cd` komutunu çalıştır.

Diğer dağıtımlarda `.tar.gz` dosyasını çift tıklayıp bir klasöre çıkar, o
klasörde bir terminal aç ve `./install.sh` yaz.

Kurulum, egzersizlerin ihtiyaç duyduğu her şeyi de kurar: NGINX, curl, Python ve
Docker. Docker'ı da başlatır.

## 3. Aç

Uygulama menüsünde "Nginxlings"i bul ve aç ya da bir terminal açıp
`nginxlings` yaz. Program ilk açılışta egzersizleri ev klasöründeki
`nginxlings` klasörüne koyar. Egzersiz dosyalarını orada düzenlersin.
Sağ paneldeki "Dosyalar:" satırı seçili egzersizin klasörünü gösterir, `o` tuşu
da o klasörü dosya yöneticisinde açar.

Ekranın üst kısmında nginxlings, en altında ise çift çizgilerle çerçevelenmiş bir
terminal bulunur. Üstteki çizgi tuşları, alttaki çizgi terminalin bulunduğu
klasörü gösterir; terminal klavyeyi aldığında çizgilerdeki yazı turuncu olur.
Terminal, seçili egzersizin klasöründe açılır; dosyaları orada düzenleyip
komutları orada çalıştırırsın, ikinci bir pencere açman gerekmez.

- `F2` ya da fareyle tıklamak: nginxlings ile terminal arasında geç
- `Alt+↑` / `Alt+↓` ya da aradaki çizgiyi fareyle sürüklemek: terminali büyüt ya da küçült
- `t` (nginxlings'teyken): terminali seçili egzersizin klasörüne götür
- Fareyle metin seçmek için `Shift` tuşunu basılı tutarak sürükle.

Temel tuşlar:

- `W` / `S`: egzersiz seç
- `c`: çözümünü kontrol et
- `h`: ipucu
- `Tab`: talimatlarla son kontrol çıktısı arasında geç
- `?`: bütün tuşlar
- `q`: çık (alttaki terminal de kapanır)

Listede `✓` kontrolü geçen egzersizi, `x` henüz geçmeyeni gösterir. Hiç kontrol
etmediğin egzersizler silik görünür.

## 4. Altta "! Kurulum" yazıyorsa

Bir şey eksikse durum satırında `! Kurulum` uyarısı görünür. `d` tuşuna bas:
Nginxlings neyin eksik olduğunu ve nasıl düzeleceğini yazar. `f` tuşu
gösterilen komutu senin için çalıştırır ve terminalde şifreni sorar.

En sık görülen durum, kullanıcının Docker'ı kullanma izninin olmamasıdır. `f`
seni `docker` grubuna ekler. Bundan sonra **oturumu kapatıp yeniden açman** (ya
da bilgisayarı yeniden başlatman) gerekir; ardından Nginxlings'i yeniden aç.

## Rocky, AlmaLinux ve RHEL hakkında

Bu dağıtımlarda kurulum Docker yerine Podman'ın `docker` komutunu kurar.
Egzersiz 10 dışındaki bütün egzersizler onunla çalışır. Egzersiz 10, Docker'ın
`127.0.0.11` adresindeki DNS sunucusunu öğretir; Podman'da bu sunucu yoktur.
Kurulum kontrolü (`d`) bunu bir not olarak gösterir, egzersiz 10'un kontrolü de
aynı açıklamayla durur. O egzersiz için eğitmeninden Docker CE kurulumunda yardım
iste.

## Kaldırma

Paketi kaldırmak için `sudo apt remove nginxlings`, `sudo dnf remove nginxlings`
ya da `sudo pacman -R nginxlings` yaz; `install.sh` ile kurduysan
`./install.sh --uninstall` çalıştır. Ev klasöründeki `nginxlings` klasörü, yani
senin çalışmaların, silinmez.
