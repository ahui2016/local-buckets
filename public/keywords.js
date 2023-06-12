$("title").text("Keywords (關鍵詞清單) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Keywords (關鍵詞清單)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("/files.html", { text: "Files" }),
        " | ",
        MJBS.createLinkElem("/pics.html", { text: "Pics" }),
        " | ",
        MJBS.createLinkElem("/buckets.html", { text: "Buckets" })
      )
  );

const PageConfig = {};

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const KeywordsList = cc("div");

function KeywordsItem(kw, i) {
  const link = MJBS.createLinkElem("#", { text: kw });
  return cc("li", {
    id: "Keywords-" + i,
    children: [link],
  });
}

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("mt-3 mb-5"),
    m(PageLoading).addClass("my-5"),
    m(PageAlert).addClass("my-3"),
    m(KeywordsList).addClass("my-3"),
    bottomDot
  );

init();

function init() {
  initFilesLimit();
}

function keywordsToItems(kwList) {
  const allKeywords = [];
  for (let i = 0; i < kwList.length; i++) {
    if (kwList[i] == "") continue;
    allKeywords.push(KeywordsItem(kwList[i], i));
  }
  return allKeywords;
}

function initFilesLimit() {
  axiosGet({
    url: "/api/auto-get-keywords",
    alert: PageAlert,
    onSuccess: (resp) => {
      const items = keywordsToItems(resp.data);
      if (items.length > 0) {
        MJBS.appendToList(KeywordsList, items);
      } else {
        PageAlert.insert("warning", "未找到任何關鍵詞.");
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
