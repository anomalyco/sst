<script setup>
  const file = ref(null);
  const { data } = await useFetch('/api/presigned');

  async function onSubmit() {
    const upload = file.value.files[0];
    const image = await fetch(data.value, {
      body: upload,
      method: "PUT",
      headers: {
        "Content-Type": upload.type,
        "Content-Disposition": `attachment; filename="${upload.name}"`,
      },
    });

    window.location.href = image.url.split("?")[0];
  }
</script>
<template>
  <form novalidate @submit.prevent="onSubmit">
    <input type="file" ref="file" accept="image/png, image/jpeg" />
    <button type="submit">Upload</button>
  </form>
</template>
