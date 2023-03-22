const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Create a Bucket (新建倉庫)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

const PageAlert = MJBS.createAlert();

const BucketIDInput = MJBS.createInput("text", "required");
const BucketEncryptBox = MJBS.createInput("checkbox");
const CreateBucketBtn = MJBS.createButton("Create", "primary");

const CreateBucketForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(
      BucketIDInput,
      "Bucket ID",
      "倉庫ID, 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)"
    ),
    MJBS.createFormCheck(BucketEncryptBox, "Secret Bucket", "是否設為加密倉庫"),
    MJBS.hiddenButtonElem(),
    m(CreateBucketBtn).on("click", (event) => {
      event.preventDefault();
      const bucketID = BucketIDInput.val();
      const encrypted = BucketEncryptBox.isChecked();
      if (!bucketID) {
        PageAlert.insert('warning', '請填寫 Bucket ID');
        return;
      }
      MJBS.disable(CreateBucketBtn);  // --------------------- disable
      axiosPost({
        url: "/api/create-bucket",
        body: {
          id: bucketID,
          encrypted: encrypted
        },
        alert: PageAlert,
        onSuccess: (resp) => {
          const bucket = resp.data;
          // window.location.href = `edit-bucket.html?id=${bucket.id}&new=true`;
          PageAlert.insert('success', JSON.stringify(bucket));
        },
        onAlways: () => {
          MJBS.enable(CreateBucketBtn);  // --------------------- enable
        }
      });

    }),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(CreateBucketForm).addClass("my-5")
  );

init();

function init() {
  MJBS.focus(BucketIDInput);
}
